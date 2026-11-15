package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	tbilisi = "ქალაქი თბილისი"
	gori    = "გორის მუნიციპალიტეტი"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

type fakeSource struct {
	mu        sync.Mutex
	nearby    []domain.NearbyParcel
	matches   []app.AddressMatch
	centreErr error
	stale     map[app.AddressRef]bool
	centres   []app.AddressRef
}

func (f *fakeSource) Parcel(_ context.Context, code domain.CadastralCode) (app.ParcelRead, error) {
	return app.ParcelRead{Parcel: domain.Parcel{Code: code}, ReadAt: clock()}, nil
}

func (f *fakeSource) ParcelsNear(context.Context, domain.Point) (app.NearbyRead, error) {
	return app.NearbyRead{Parcels: f.nearby, ReadAt: clock()}, nil
}

func (f *fakeSource) MatchAddresses(_ context.Context, text string) (app.MatchRead, error) {
	if text == "zzqxv" {
		return app.MatchRead{ReadAt: clock()}, nil
	}

	return app.MatchRead{Matches: f.matches, ReadAt: clock()}, nil
}

func (f *fakeSource) AddressCentre(_ context.Context, ref app.AddressRef) (app.CentreRead, error) {
	f.mu.Lock()
	f.centres = append(f.centres, ref)
	f.mu.Unlock()

	if f.centreErr != nil {
		return app.CentreRead{}, f.centreErr
	}

	readAt := clock()
	if f.stale[ref] {
		readAt = readAt.Add(-time.Hour)
	}

	return app.CentreRead{
		Centre: domain.Point{Lat: 41.7, Lon: 44.8},
		ReadAt: readAt,
		Stale:  f.stale[ref],
	}, nil
}

func nearbyParcels(t *testing.T) []domain.NearbyParcel {
	t.Helper()

	codes := []string{
		"01.15.05.066.020", "01.15.05.066.015", "01.15.04.028.015", "01.15.04.028.003",
		"01.15.04.026.012", "01.15.04.027.019", "01.15.04.027.016", "01.15.04.023.007",
		"01.15.04.023.004", "01.15.04.019.004", "01.15.04.019.006", "01.15.04.020.004",
		"01.15.04.007.010", "01.15.04.007.014", "01.15.04.007.024",
	}
	distances := []int{0, 92, 101, 103, 156, 297, 343, 410, 516, 566, 621, 650, 724, 787, 797}

	parcels := make([]domain.NearbyParcel, 0, len(codes))

	for i, text := range codes {
		code, err := domain.ParseCadastralCode(text)
		if err != nil {
			t.Fatal(err)
		}

		parcels = append(parcels, domain.NearbyParcel{Code: code, DistanceM: distances[i]})
	}

	return parcels
}

// NAPRPARC-SC-012: The nearest parcels come first, cut to the limit.
func TestListNearby(t *testing.T) {
	t.Parallel()

	source := &fakeSource{nearby: nearbyParcels(t)}
	list := app.NewListNearby(source)
	point := domain.Point{Lat: 41.6934, Lon: 44.8015}

	five, err := list.List(t.Context(), app.NearbyQuery{Point: point, Limit: 5})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(five.Parcels) != 5 {
		t.Fatalf("len(Parcels) = %d, want 5", len(five.Parcels))
	}

	if got := five.Parcels[0]; got.Code.String() != "01.15.05.066.020" || got.DistanceM != 0 {
		t.Errorf("Parcels[0] = %+v, want 01.15.05.066.020 at 0", got)
	}

	if got := five.Parcels[1]; got.Code.String() != "01.15.05.066.015" || got.DistanceM != 92 {
		t.Errorf("Parcels[1] = %+v, want 01.15.05.066.015 at 92", got)
	}

	all, err := list.List(t.Context(), app.NearbyQuery{Point: point, Limit: 20})
	if err != nil || len(all.Parcels) != 15 {
		t.Errorf("List(20) = %d parcels, %v, want 15", len(all.Parcels), err)
	}
}

func addressMatches() []app.AddressMatch {
	municipalities := []string{
		gori, gori, "თელავის მუნიციპალიტეტი",
		"თელავის მუნიციპალიტეტი", tbilisi, tbilisi, tbilisi, tbilisi, tbilisi,
		"ქალაქი ქუთაისი",
	}

	matches := make([]app.AddressMatch, 0, len(municipalities))
	for i, municipality := range municipalities {
		matches = append(matches, app.AddressMatch{
			Text:         fmt.Sprintf("ილია ჭავჭავაძის გამზირი, N %d", 40+i),
			Municipality: municipality,
			Ref:          app.AddressRef(fmt.Sprintf("ract_g_borders:A%d", i)),
		})
	}

	return matches
}

// NAPRPARC-SC-013: A municipality keeps its matches. The use case reads a centre for each
// address it returns, and none for an empty search.
func TestSearchAddresses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		query            app.AddressQuery
		wantFirst        string
		wantCount        int
		wantMunicipality string
	}{
		{
			name: "tbilisi",
			query: app.AddressQuery{
				Text:         "ჭავჭავაძის გამზირი 44",
				Municipality: " თბილისი ",
				Limit:        5,
			},
			wantFirst:        "ილია ჭავჭავაძის გამზირი, N 44",
			wantCount:        5,
			wantMunicipality: tbilisi,
		},
		{
			name:             "every municipality",
			query:            app.AddressQuery{Text: "ჭავჭავაძის გამზირი 44", Limit: 5},
			wantFirst:        "ილია ჭავჭავაძის გამზირი, N 40",
			wantCount:        5,
			wantMunicipality: gori,
		},
		{name: "none", query: app.AddressQuery{Text: "zzqxv", Limit: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checkSearch(t, tt.query, tt.wantFirst, tt.wantMunicipality, tt.wantCount)
		})
	}
}

func checkSearch(
	t *testing.T,
	query app.AddressQuery,
	wantFirst, wantMunicipality string,
	wantCount int,
) {
	t.Helper()

	source := &fakeSource{matches: addressMatches()}

	list, err := app.NewSearchAddresses(source).Search(t.Context(), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(list.Addresses) != wantCount || len(source.centres) != wantCount {
		t.Fatalf(
			"addresses, centres = %d, %d, want %d",
			len(list.Addresses),
			len(source.centres),
			wantCount,
		)
	}

	if wantCount == 0 {
		return
	}

	first := list.Addresses[0]
	if first.Text != wantFirst || first.Municipality != wantMunicipality {
		t.Errorf("Addresses[0] = %+v, want %s in %s", first, wantFirst, wantMunicipality)
	}

	if first.Centre != (domain.Point{Lat: 41.7, Lon: 44.8}) {
		t.Errorf("Addresses[0].Centre = %v, want 41.7, 44.8", first.Centre)
	}
}

// NAPRPARC-SC-013: The list carries the oldest read and the stale mark of any centre. A
// failed centre fails the search.
func TestSearchAddressesJoinsTheReads(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		matches: addressMatches(),
		stale:   map[app.AddressRef]bool{"ract_g_borders:A1": true},
	}

	list, err := app.NewSearchAddresses(source).
		Search(t.Context(), app.AddressQuery{Text: "x", Limit: 3})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if !list.Stale || !list.ReadAt.Equal(clock().Add(-time.Hour)) {
		t.Errorf("Stale, ReadAt = %v, %v, want true and one hour before", list.Stale, list.ReadAt)
	}

	if diff := cmp.Diff(
		[]app.AddressRef{"ract_g_borders:A0", "ract_g_borders:A1", "ract_g_borders:A2"},
		sorted(source.centres),
	); diff != "" {
		t.Errorf("centres (-want +got):\n%s", diff)
	}

	failing := &fakeSource{matches: addressMatches(), centreErr: kernel.ErrSourceUnavailable}

	_, err = app.NewSearchAddresses(failing).
		Search(t.Context(), app.AddressQuery{Text: "x", Limit: 3})
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("error = %v, want ErrSourceUnavailable", err)
	}
}

// NAPRPARC-SC-012: GetParcel passes the code to the source, and its error back.
func TestGetParcel(t *testing.T) {
	t.Parallel()

	code, err := domain.ParseCadastralCode("01.15.05.066.020")
	if err != nil {
		t.Fatal(err)
	}

	read, err := app.NewGetParcel(&fakeSource{}).Get(t.Context(), code)
	if err != nil || read.Parcel.Code != code {
		t.Errorf("Get() = %+v, %v, want the parcel %s", read, err, code)
	}
}

func sorted(refs []app.AddressRef) []app.AddressRef {
	refs = slices.Clone(refs)
	slices.Sort(refs)

	return refs
}
