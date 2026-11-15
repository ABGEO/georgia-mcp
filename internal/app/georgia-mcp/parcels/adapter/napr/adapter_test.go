package napr_test

import (
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	napradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/napr"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	naprclient "github.com/abgeo/georgia-mcp/pkg/napr"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata    = "../../../../../../pkg/napr/testdata/"
	freedomCode = "01.15.05.066.020"

	searchCode   = "search-code.json"
	infoFreedom  = "info-freedom.json"
	shapeFreedom = "shape-freedom.json"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 8, 0, 0, 0, time.UTC)
}

// answers names the file of each method. An empty name answers a body that is not JSON.
type answers struct {
	search, near, info, shape, addressShape string
}

func serve(t *testing.T, files answers) *napradapter.Adapter {
	t.Helper()

	read := func(file string) []byte {
		if file == "" {
			return []byte("not json")
		}

		body, err := os.ReadFile(testdata + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/map/portal/search":
			_, _ = w.Write(read(files.search))
		case r.URL.Path == "/lr/bo/mg/getinfo2":
			_, _ = w.Write(read(files.near))
		case r.URL.Path == "/ar/mg/getinfo":
			_, _ = w.Write(read(files.addressShape))
		case strings.Contains(r.URL.RawQuery, "res=shp"):
			_, _ = w.Write(read(files.shape))
		default:
			_, _ = w.Write(read(files.info))
		}
	}))
	t.Cleanup(server.Close)

	return napradapter.NewAdapter(naprclient.New(vendorhttp.WithBaseURL(server.URL)), clock)
}

func code(t *testing.T, text string) domain.CadastralCode {
	t.Helper()

	parsed, err := domain.ParseCadastralCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// freedom answers the parcel of Freedom Square. Each change replaces one file.
func freedom(change func(*answers)) answers {
	files := answers{search: searchCode, info: infoFreedom, shape: shapeFreedom}
	if change != nil {
		change(&files)
	}

	return files
}

func readFreedom(t *testing.T) app.ParcelRead {
	t.Helper()

	read, err := serve(t, freedom(nil)).Parcel(t.Context(), code(t, freedomCode))
	if err != nil {
		t.Fatalf("Parcel() error = %v", err)
	}

	return read
}

// NAPRPARC-SC-009: The recorded parcel of Freedom Square converts to the domain, with the
// texts of NAPR.
func TestParcel(t *testing.T) {
	t.Parallel()

	parcel := readFreedom(t).Parcel

	type summary struct {
		Code, Address string
		Area          domain.Area
		LandType      domain.LandType
		Ownership     domain.OwnershipType
		Owners        []string
	}

	got := summary{
		Code:      parcel.Code.String(),
		Address:   parcel.Address,
		Area:      parcel.Area,
		LandType:  parcel.LandType.Code,
		Ownership: parcel.Ownership.Code,
		Owners:    parcel.Owners,
	}
	want := summary{
		Code:      freedomCode,
		Address:   "ქალაქი თბილისი , მოედანი თავისუფლება , (ნაკვეთი 66/020)",
		Area:      domain.Area{Value: "618", Unit: "m2"},
		LandType:  domain.LandNonAgricultural,
		Ownership: domain.Ownership,
		Owners:    []string{"ქალაქ თბილისის თვითმმართველი ერთეული"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parcel() (-want +got):\n%s", diff)
	}
}

// NAPRPARC-SC-009: The documents come sorted with their links, the centre is on the
// square, and the read is in Tbilisi.
func TestParcelDocumentsAndCentre(t *testing.T) {
	t.Parallel()

	read := readFreedom(t)
	documents := read.Parcel.Documents

	if len(documents) != 2 || documents[0].Kind.Code != domain.RegistryExtract ||
		documents[0].Date.String() != "2009-08-14" {
		t.Errorf("Documents = %+v, want the extract of 2009-08-14 first", documents)
	}

	if !strings.HasPrefix(documents[0].Link, "https://bs.napr.gov.ge/GetBlob?pid=101&bid=") {
		t.Errorf("Documents[0].Link = %q, want a link of bs.napr.gov.ge", documents[0].Link)
	}

	if metres(read.Parcel.Centre, domain.Point{Lat: 41.69340, Lon: 44.80151}) > 2 {
		t.Errorf("Centre = %v, want within 2 m of 41.69340, 44.80151", read.Parcel.Centre)
	}

	if read.MapURL != "https://maps.gov.ge/map/portal/#search/result/getinfo?lbl=lr_parcels:AAArWyAATAAIs9BAAB" {
		t.Errorf("MapURL = %q, want the page of the label", read.MapURL)
	}

	if !read.ReadAt.Equal(clock()) ||
		read.ReadAt.Location().String() != kernel.TbilisiZone().String() {
		t.Errorf("ReadAt = %v, want %v in Tbilisi", read.ReadAt, clock())
	}
}

func metres(a, b domain.Point) float64 {
	const degree = 111_320.0

	dLat := (a.Lat - b.Lat) * degree
	dLon := (a.Lon - b.Lon) * degree * math.Cos(a.Lat*math.Pi/180)

	return math.Hypot(dLat, dLon)
}

// NAPRPARC-SC-011: An empty detail, a broken body, a broken shape, or a detail of another
// code is a failure of NAPR. An empty search is no parcel.
func TestParcelFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files answers
		asked string
		want  error
	}{
		{
			name:  "empty detail",
			files: freedom(func(a *answers) { a.info = "info-empty.json" }),
			want:  kernel.ErrSourceUnavailable,
		},
		{
			name:  "body",
			files: freedom(func(a *answers) { a.info = "" }),
			want:  kernel.ErrSourceUnavailable,
		},
		{
			name:  "shape",
			files: freedom(func(a *answers) { a.shape = infoFreedom }),
			want:  kernel.ErrSourceUnavailable,
		},
		{
			name:  "another code",
			files: freedom(func(a *answers) { a.info = "info-coowned.json" }),
			want:  kernel.ErrSourceUnavailable,
		},
		{
			name:  "no parcel",
			files: answers{search: "search-empty.json"},
			want:  app.ErrParcelNotFound,
		},
		{
			name:  "a result of another code",
			files: answers{search: searchCode},
			asked: "01.15.05.066.015",
			want:  app.ErrParcelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			asked := tt.asked
			if asked == "" {
				asked = freedomCode
			}

			_, err := serve(t, tt.files).Parcel(t.Context(), code(t, asked))
			if !errors.Is(err, tt.want) {
				t.Errorf("Parcel() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// NAPRPARC-SC-010: The detail of a co-owned parcel with no owner converts with no error.
func TestParcelNoOwnerDetail(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(testdata + searchCode)
	if err != nil {
		t.Fatal(err)
	}

	search := strings.ReplaceAll(string(body), freedomCode, "01.15.05.066.015")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := "info-no-owner.json"

		switch {
		case r.URL.Path == "/map/portal/search":
			_, _ = w.Write([]byte(search))

			return
		case strings.Contains(r.URL.RawQuery, "res=shp"):
			file = shapeFreedom
		}

		answer, _ := os.ReadFile(testdata + file)
		_, _ = w.Write(answer)
	}))
	t.Cleanup(server.Close)

	adapter := napradapter.NewAdapter(naprclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	read, err := adapter.Parcel(t.Context(), code(t, "01.15.05.066.015"))
	if err != nil {
		t.Fatalf("Parcel() error = %v", err)
	}

	if read.Parcel.Ownership.Code != domain.CoOwnership || len(read.Parcel.Owners) != 0 ||
		len(read.Parcel.Documents) != 0 {
		t.Errorf("Parcel = %+v, want co_ownership with no owner and no document", read.Parcel)
	}
}

// NAPRPARC-SC-012: The parcels near Freedom Square keep the order of NAPR, with whole
// metres.
func TestParcelsNear(t *testing.T) {
	t.Parallel()

	read, err := serve(t, answers{near: "near-freedom.json"}).
		ParcelsNear(t.Context(), domain.Point{Lat: 41.6934, Lon: 44.8015})
	if err != nil {
		t.Fatalf("ParcelsNear() error = %v", err)
	}

	if len(read.Parcels) != 15 {
		t.Fatalf("len(Parcels) = %d, want 15", len(read.Parcels))
	}

	second := read.Parcels[1]
	if second.Code.String() != "01.15.05.066.015" || second.DistanceM != 92 {
		t.Errorf("Parcels[1] = %+v, want 01.15.05.066.015 at 92", second)
	}

	if !strings.Contains(read.Parcels[3].Address, `"ცეკავშირის"`) {
		t.Errorf("Parcels[3].Address = %q, want the quoted name", read.Parcels[3].Address)
	}
}

// NAPRPARC-SC-013: The address search keeps the addresses, with their municipality and
// reference.
func TestMatchAddresses(t *testing.T) {
	t.Parallel()

	adapter := serve(t, answers{search: "search-address.json", addressShape: "shape-address.json"})

	read, err := adapter.MatchAddresses(t.Context(), "ჭავჭავაძის გამზირი 44")
	if err != nil {
		t.Fatalf("MatchAddresses() error = %v", err)
	}

	if len(read.Matches) != 10 {
		t.Fatalf("len(Matches) = %d, want 10", len(read.Matches))
	}

	first := read.Matches[0]
	if first.Text != "ქალაქი გორი, ილია ჭავჭავაძის გამზირი, N 44" ||
		first.Municipality != "გორის მუნიციპალიტეტი" ||
		first.Ref != "ract_g_borders:AAAlnvAAvAADyJ1AAF" {
		t.Errorf("Matches[0] = %+v", first)
	}
}

// NAPRPARC-SC-013: The centre of an address is inside its multipolygon.
func TestAddressCentre(t *testing.T) {
	t.Parallel()

	adapter := serve(t, answers{addressShape: "shape-address.json"})

	read, err := adapter.AddressCentre(t.Context(), "ract_g_borders:AAAlnvAAyAAEGARAAD")
	if err != nil {
		t.Fatalf("AddressCentre() error = %v", err)
	}

	if read.Centre.Lat < 41.709 || read.Centre.Lat > 41.710 ||
		read.Centre.Lon < 44.765 || read.Centre.Lon > 44.767 {
		t.Errorf("Centre = %v, want inside the multipolygon of N 44", read.Centre)
	}
}

// NAPRPARC-SC-013: A parcel result of a search is not an address.
func TestMatchAddressesLeavesOutAParcel(t *testing.T) {
	t.Parallel()

	read, err := serve(
		t,
		answers{search: searchCode},
	).MatchAddresses(t.Context(), freedomCode)
	if err != nil || len(read.Matches) != 0 {
		t.Errorf("MatchAddresses() = %+v, %v, want no address", read.Matches, err)
	}
}
