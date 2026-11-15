package napr_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/napr"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const freedomCode = "01.15.05.066.020"

type recorded struct {
	method string
	uri    string
	header http.Header
	body   string
}

type requestLog struct {
	mu       sync.Mutex
	requests []recorded
}

func (l *requestLog) all() []recorded {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.requests)
}

// serve answers each request with the file and the status, and records the request.
func serve(t *testing.T, file string, status int) (*napr.Client, *requestLog) {
	t.Helper()

	body := []byte("not json")

	if file != "" {
		read, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		body = read
	}

	requests := &requestLog{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()

		requests.mu.Lock()
		requests.requests = append(requests.requests, recorded{
			method: r.Method,
			uri:    r.URL.RequestURI(),
			header: r.Header.Clone(),
			body:   r.PostForm.Encode(),
		})
		requests.mu.Unlock()

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return napr.New(vendorhttp.WithBaseURL(server.URL)), requests
}

// NAPRPARC-SC-001: Each method sends the User-Agent of the server, and a label with no
// encoding.
func TestMethodsSendTheRequest(t *testing.T) {
	t.Parallel()

	for _, tt := range requestCases(t) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, requests := serve(t, tt.file, http.StatusOK)

			if err := tt.call(client); err != nil {
				t.Fatalf("error = %v", err)
			}

			got := requests.all()[0]
			if got.method != tt.method || got.uri != tt.uri {
				t.Errorf("request = %s %s, want %s %s", got.method, got.uri, tt.method, tt.uri)
			}

			if got.body != tt.body {
				t.Errorf("body = %q, want %q", got.body, tt.body)
			}

			agent := got.header.Get("User-Agent")
			if !strings.HasPrefix(agent, "georgia-mcp/") ||
				!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
				t.Errorf("User-Agent = %q, want the agent of Georgia MCP", agent)
			}
		})
	}
}

type requestCase struct {
	name   string
	file   string
	call   func(*napr.Client) error
	method string
	uri    string
	body   string
}

//nolint:funlen // One case for each method.
func requestCases(t *testing.T) []requestCase {
	t.Helper()

	const label = "lr_parcels:AAArWyAAKAAH/11AAI"

	return []requestCase{
		{
			name: "Search",
			file: "search-code.json",
			call: func(c *napr.Client) error {
				_, err := c.Search(t.Context(), freedomCode)

				return err
			},
			method: http.MethodPost,
			uri:    "/map/portal/search",
			body:   "keyword=" + freedomCode + "&keyword_description=",
		},
		{
			name: "ParcelsAt",
			file: "near-outside.json",
			call: func(c *napr.Client) error {
				_, err := c.ParcelsAt(t.Context(), 41.6934, 44.8015)

				return err
			},
			method: http.MethodGet,
			uri: "/lr/bo/mg/getinfo2?C=0&FRAME_NAME=BY_COORDS_AND_COMPASS.GETINFO.MAPGOV" +
				"&LAT=41.693400&LON=44.801500",
		},
		{
			name: "Parcel",
			file: "info-freedom.json",
			call: func(c *napr.Client) error {
				_, err := c.Parcel(t.Context(), label)

				return err
			},
			method: http.MethodGet,
			uri:    "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAAKAAH/11AAI&fmt=json&lang=ka",
		},
		{
			name: "ParcelShape",
			file: "shape-freedom.json",
			call: func(c *napr.Client) error {
				_, err := c.ParcelShape(t.Context(), label)

				return err
			},
			method: http.MethodGet,
			uri:    "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAAKAAH/11AAI&res=shp",
		},
		{
			name: "AddressShape",
			file: "shape-address.json",
			call: func(c *napr.Client) error {
				_, err := c.AddressShape(t.Context(), "ract_g_borders:AAAlnvAAyAAEGARAAD")

				return err
			},
			method: http.MethodGet,
			uri:    "/ar/mg/getinfo?lbl=ract_g_borders:AAAlnvAAyAAEGARAAD&res=shp",
		},
	}
}

// NAPRPARC-SC-001: A label that is not a label, or an empty keyword, sends no call.
func TestMethodsRefuseABadInput(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "info-freedom.json", http.StatusOK)

	for _, label := range []string{"", "lr_parcels:AA&x=1", "lr_parcels:AA AA", "AAArWyAATAAIs9BAAB"} {
		if _, err := client.Parcel(t.Context(), label); !errors.Is(err, napr.ErrInvalidLabel) {
			t.Errorf("Parcel(%q) error = %v, want ErrInvalidLabel", label, err)
		}
	}

	if _, err := client.Search(t.Context(), " "); !errors.Is(err, napr.ErrEmptyKeyword) {
		t.Errorf("Search(\" \") error = %v, want ErrEmptyKeyword", err)
	}

	if n := len(requests.all()); n != 0 {
		t.Errorf("calls = %d, want 0", n)
	}
}

// NAPRPARC-SC-003: The recorded searches give a parcel, ten addresses, or nothing.
func TestSearchReadsTheAnswer(t *testing.T) {
	t.Parallel()

	t.Run("code", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "search-code.json", http.StatusOK)

		results, err := client.Search(t.Context(), freedomCode)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		want := napr.SearchResult{
			Name:     freedomCode,
			Descript: "ქალაქი თბილისი , მოედანი თავისუფლება ,   (ნაკვეთი 66/020)",
			Details: napr.SearchDetails{
				Type:         "address",
				InfoLink:     "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB",
				GeometryLink: "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB&res=shp",
			},
		}
		if len(results) != 1 || results[0] != want {
			t.Errorf("results = %+v, want [%+v]", results, want)
		}
	})

	t.Run("address", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "search-address.json", http.StatusOK)

		results, err := client.Search(t.Context(), "ჭავჭავაძის გამზირი 44")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) != 10 {
			t.Fatalf("len(results) = %d, want 10", len(results))
		}

		if results[0].Descript != "გორის მუნიციპალიტეტი" {
			t.Errorf("results[0].Descript = %q, want გორის მუნიციპალიტეტი", results[0].Descript)
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "search-empty.json", http.StatusOK)

		results, err := client.Search(t.Context(), "99.99.99.999.999")
		if err != nil || len(results) != 0 {
			t.Errorf("Search() = %v, %v, want no result and no error", results, err)
		}
	})
}

// NAPRPARC-SC-003: A status that is not a success is an *APIError, and a body that is not
// JSON is an error.
func TestMethodsFail(t *testing.T) {
	t.Parallel()

	t.Run("status", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "search-empty.json", http.StatusInternalServerError)

		_, err := client.Search(t.Context(), freedomCode)

		var apiErr *napr.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
			t.Fatalf("error = %v, want an *APIError with 500", err)
		}

		if !errors.Is(err, napr.ErrUnexpectedStatus) {
			t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false, want true")
		}
	})

	t.Run("body", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "", http.StatusOK)

		if _, err := client.ParcelsAt(t.Context(), 41.6934, 44.8015); err == nil {
			t.Error("ParcelsAt() error = nil, want an error")
		}

		if _, err := client.Parcel(t.Context(), "lr_parcels:AAArWyAATAAIs9BAAB"); err == nil {
			t.Error("Parcel() error = nil, want an error")
		}
	})
}

// NAPRPARC-SC-004: ParcelsAt reads an answer whose address holds a quotation mark that
// the source does not escape.
func TestParcelsAtRepairsTheAnswer(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "near-freedom.json", http.StatusOK)

	parcels, err := client.ParcelsAt(t.Context(), 41.6934, 44.8015)
	if err != nil {
		t.Fatalf("ParcelsAt() error = %v", err)
	}

	if len(parcels) != 15 {
		t.Fatalf("len(parcels) = %d, want 15", len(parcels))
	}

	if parcels[0].Name != freedomCode || parcels[0].Distance != "0" {
		t.Errorf("parcels[0] = %+v, want 01.15.05.066.020 at 0", parcels[0])
	}

	fourth := parcels[3]
	if fourth.Name != "01.15.04.028.003" || !strings.Contains(fourth.Descript, `"ცეკავშირის"`) {
		t.Errorf("parcels[3] = %+v, want 01.15.04.028.003 with \"ცეკავშირის\"", parcels[3])
	}

	if !strings.HasPrefix(parcels[0].ResultLink, "#search/result/getinfo?lbl=lr_parcels:") {
		t.Errorf("parcels[0].ResultLink = %q, want a link with a label", parcels[0].ResultLink)
	}
}

// NAPRPARC-SC-009: Parcel and ParcelShape read the recorded parcel of Freedom Square.
func TestParcelReadsTheAnswer(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "info-freedom.json", http.StatusOK)

	infos, err := client.Parcel(t.Context(), "lr_parcels:AAArWyAATAAIs9BAAB")
	if err != nil {
		t.Fatalf("Parcel() error = %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1", len(infos))
	}

	info := infos[0]

	want := napr.ParcelDetail{
		CadCode:      freedomCode,
		AreaMeasured: "618",
		Unit:         "კვ.მ.",
		Address:      "ქალაქი თბილისი , მოედანი თავისუფლება ,   (ნაკვეთი 66/020)",
		LandType:     "არასასოფლო სამეურნეო",
		OwnTypeName:  "საკუთრება",
	}
	if info.Detail != want {
		t.Errorf("Detail = %+v, want %+v", info.Detail, want)
	}

	if len(info.Owners) != 1 || len(info.Docs) != 2 ||
		info.Docs[0].List[0].DocDate != "13.08.2009" {
		t.Errorf("owners, docs = %+v, %+v, want 1 owner and 2 documents", info.Owners, info.Docs)
	}

	shapes, _ := serve(t, "shape-freedom.json", http.StatusOK)

	got, err := shapes.ParcelShape(t.Context(), "lr_parcels:AAArWyAATAAIs9BAAB")
	if err != nil || len(got) != 1 || !strings.HasPrefix(got[0].Shape, "POLYGON ((44.80166") {
		t.Errorf("ParcelShape() = %+v, %v, want one POLYGON", got, err)
	}
}

// NAPRPARC-SC-003: Label reads the label of a link, and refuses a link with none.
func TestLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		link string
		want string
		ok   bool
	}{
		{
			"/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB",
			"lr_parcels:AAArWyAATAAIs9BAAB",
			true,
		},
		{
			"/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB&res=shp",
			"lr_parcels:AAArWyAATAAIs9BAAB",
			true,
		},
		{
			"#search/result/getinfo?lbl=lr_parcels:AAArWyAAKAAH/11AAI",
			"lr_parcels:AAArWyAAKAAH/11AAI",
			true,
		},
		{
			"/ar/mg/getinfo?lbl=ract_g_borders:AAAlnvAAyAAEGARAAD",
			"ract_g_borders:AAAlnvAAyAAEGARAAD",
			true,
		},
		{"/map/portal/search", "", false},
		{"/x?lbl=AA%20B", "", false},
	}

	for _, tt := range tests {
		got, ok := napr.Label(tt.link)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Label(%q) = %q, %v, want %q, %v", tt.link, got, ok, tt.want, tt.ok)
		}
	}
}
