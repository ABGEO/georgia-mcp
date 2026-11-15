package domain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	freedomCode  = "01.15.05.066.020"
	squareMetres = "კვ.მ."

	registryExtract = "ამონაწერი საჯარო რეესტრიდან"
)

// NAPRPARC-SC-005: A cadastral code has 4 or 5 groups of digits. A longer code names a
// unit and gives the code of its parcel.
func TestParseCadastralCode(t *testing.T) {
	t.Parallel()

	valid := []struct{ text, want string }{
		{freedomCode, freedomCode},
		{" 53.20.41.012 ", "53.20.41.012"},
		{"05.31.07.1000", "05.31.07.1000"},
	}
	for _, tt := range valid {
		text, want := tt.text, tt.want

		code, err := domain.ParseCadastralCode(text)
		if err != nil || code.String() != want {
			t.Errorf("ParseCadastralCode(%q) = %q, %v, want %q", text, code, err, want)
		}
	}

	for _, text := range []string{"01.15.05", "01.15.05.066.02a", "", "01..05.066.020", "01.15.05.066.020."} {
		if _, err := domain.ParseCadastralCode(
			text,
		); !errors.Is(
			err,
			domain.ErrInvalidCadastralCode,
		) {
			t.Errorf("ParseCadastralCode(%q) error = %v, want ErrInvalidCadastralCode", text, err)
		}
	}

	_, err := domain.ParseCadastralCode("01.14.11.005.029.01.501")

	var unit *domain.UnitCodeError
	if !errors.As(err, &unit) || unit.Parcel.String() != "01.14.11.005.029" {
		t.Fatalf("error = %v, want a *UnitCodeError with 01.14.11.005.029", err)
	}

	if !errors.Is(err, domain.ErrInvalidCadastralCode) {
		t.Errorf("errors.Is(err, ErrInvalidCadastralCode) = false, want true")
	}
}

// NAPRPARC-SC-005: A valid code gives itself from String.
func FuzzParseCadastralCode(f *testing.F) {
	for _, seed := range []string{freedomCode, "53.20.41.012", "01.14.11.005.029.01.501", "", "1.2.3.4"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		code, err := domain.ParseCadastralCode(text)
		if err != nil {
			return
		}

		again, err := domain.ParseCadastralCode(code.String())
		if err != nil || again != code {
			t.Errorf("ParseCadastralCode(%q) = %q, and again %q, %v", text, code, again, err)
		}
	})
}

// NAPRPARC-SC-006: A point outside the box of Georgia, or with a swapped order, fails.
func TestNewPoint(t *testing.T) {
	t.Parallel()

	point, err := domain.NewPoint(41.6934, 44.8015)
	if err != nil || point != (domain.Point{Lat: 41.6934, Lon: 44.8015}) {
		t.Errorf("NewPoint(41.6934, 44.8015) = %v, %v", point, err)
	}

	for _, bad := range [][2]float64{{44.8015, 41.6934}, {48.85, 2.35}, {math.NaN(), 44.8}, {41.7, math.Inf(1)}} {
		if _, err := domain.NewPoint(
			bad[0],
			bad[1],
		); !errors.Is(
			err,
			domain.ErrPointOutsideGeorgia,
		) {
			t.Errorf(
				"NewPoint(%v, %v) error = %v, want ErrPointOutsideGeorgia",
				bad[0],
				bad[1],
				err,
			)
		}
	}
}

// NAPRPARC-SC-006: A point never panics, and a valid point is inside the box.
func FuzzNewPoint(f *testing.F) {
	f.Add(41.6934, 44.8015)
	f.Add(44.8015, 41.6934)
	f.Add(math.NaN(), 0.0)

	f.Fuzz(func(t *testing.T, lat, lon float64) {
		point, err := domain.NewPoint(lat, lon)
		if err != nil {
			return
		}

		if point.Lat < 41 || point.Lat > 43.6 || point.Lon < 40 || point.Lon > 46.8 {
			t.Errorf("NewPoint(%v, %v) = %v, outside the box", lat, lon, point)
		}
	})
}

// NAPRPARC-SC-007: A text of the source gives its code, or other with the text.
func TestClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		got  [2]string
		want [2]string
	}{
		{
			pair(domain.ParseLandType("სასოფლო-სამეურნეო")),
			[2]string{"agricultural", "სასოფლო-სამეურნეო"},
		},
		{
			pair(domain.ParseLandType("არასასოფლო  სამეურნეო")),
			[2]string{"non_agricultural", "არასასოფლო სამეურნეო"},
		},
		{pair(domain.ParseLandType("ტყე")), [2]string{"other", "ტყე"}},
		{
			pair(domain.ParseOwnershipType(" თანასაკუთრება ")),
			[2]string{"co_ownership", "თანასაკუთრება"},
		},
		{pair(domain.ParseOwnershipType("საკუთრება")), [2]string{"ownership", "საკუთრება"}},
		{
			pair(domain.ParseDocumentKind("საკადასტრო გეგმა")),
			[2]string{"cadastral_plan", "საკადასტრო გეგმა"},
		},
		{
			pair(domain.ParseDocumentKind(registryExtract)),
			[2]string{"registry_extract", registryExtract},
		},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func pair[T ~string](classified domain.Classified[T]) [2]string {
	return [2]string{string(classified.Code), classified.Text}
}

// NAPRPARC-SC-007: An area in square metres has the unit m2. Another unit stays.
func TestNewArea(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value, unit string
		want        domain.Area
		wantErr     error
	}{
		{value: "618", unit: squareMetres, want: domain.Area{Value: "618", Unit: "m2"}},
		{value: "1.5", unit: "ჰა", want: domain.Area{Value: "1.5", Unit: "ჰა"}},
		{
			value: " 1078 ",
			unit:  " " + squareMetres + " ",
			want:  domain.Area{Value: "1078", Unit: "m2"},
		},
		{value: "abc", unit: squareMetres, wantErr: domain.ErrInvalidArea},
		{value: "", unit: squareMetres, wantErr: domain.ErrInvalidArea},
		{value: "618", unit: "", wantErr: domain.ErrInvalidArea},
	}

	for _, tt := range tests {
		got, err := domain.NewArea(tt.value, tt.unit)
		if !errors.Is(err, tt.wantErr) || got != tt.want {
			t.Errorf(
				"NewArea(%q, %q) = %v, %v, want %v, %v",
				tt.value,
				tt.unit,
				got,
				err,
				tt.want,
				tt.wantErr,
			)
		}
	}
}

// NAPRPARC-SC-008: The documents of the co-owned parcel come by kind, newest first, with
// no date last. A copy of one link is one document.
func TestSortDocuments(t *testing.T) {
	t.Parallel()

	extract := domain.ParseDocumentKind(registryExtract)
	plan := domain.ParseDocumentKind("საკადასტრო გეგმა")

	doc := func(kind domain.Classified[domain.DocumentKind], number, date string) domain.RegistryDocument {
		return document(t, kind, number, date)
	}

	first := doc(extract, "012006061599", "")
	twin := doc(extract, "882026245970", "2026-03-25")
	twin.Link += "b"

	in := []domain.RegistryDocument{
		doc(plan, "882012188938", "2012-05-04"),
		doc(plan, "882009292548", "2009-10-08"),
		first,
		doc(extract, "012006061520", ""),
		doc(extract, "012006058945", ""),
		doc(extract, "882026245970", "2026-03-25"),
		twin,
		doc(extract, "882022642402", "2022-09-07"),
		doc(extract, "012007116962", "2007-08-10"),
		doc(extract, "012006089250", "2006-12-20"),
		doc(extract, "012006084277", "2006-12-07"),
		doc(extract, "012006063566", "2006-10-19"),
		doc(extract, "882022642402", "2022-09-07"),
		first,
	}
	in[12].Link += "c"

	got := domain.SortDocuments(in)

	numbers := make([]string, 0, len(got))
	for _, document := range got {
		numbers = append(numbers, document.RegistrationNumber)
	}

	want := []string{
		"882026245970", "882026245970", "882022642402", "882022642402", "012007116962",
		"012006089250", "012006084277", "012006063566",
		"012006061599", "012006061520", "012006058945",
		"882012188938", "882009292548",
	}
	if diff := cmp.Diff(want, numbers); diff != "" {
		t.Errorf("SortDocuments() (-want +got):\n%s", diff)
	}
}

func document(
	t *testing.T,
	kind domain.Classified[domain.DocumentKind],
	number, date string,
) domain.RegistryDocument {
	t.Helper()

	doc := domain.RegistryDocument{
		Kind:               kind,
		RegistrationNumber: number,
		Link:               "https://x/" + number + date,
	}
	if date != "" {
		parsed, err := kernel.ParseDate(date)
		if err != nil {
			t.Fatal(err)
		}

		doc.Date = &parsed
	}

	return doc
}
