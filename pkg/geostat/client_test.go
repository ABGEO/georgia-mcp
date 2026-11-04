package geostat_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/geostat"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const monthlyPath = "Price Statistics/CPI/National Consumer Price Index/" +
	"CPI_in_Georgia_(previous_month=100)_orenovani2.px"

type recorded struct {
	method  string
	rawPath string
	path    string
	header  http.Header
	body    []byte
}

// serve answers each request with the file and the status, and records the request.
func serve(t *testing.T, file string, status int) (*geostat.Client, *[]recorded) {
	t.Helper()

	var body []byte

	if file != "" {
		read, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		body = read
	}

	var requests []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ := io.ReadAll(r.Body)
		requests = append(requests, recorded{
			method: r.Method, rawPath: r.URL.EscapedPath(), path: r.URL.Path,
			header: r.Header.Clone(), body: sent,
		})

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return geostat.New(vendorhttp.WithBaseURL(server.URL)), &requests
}

// GEOSTAT-SC-001: Each method sends the User-Agent of the server and asks for JSON.
func TestEachMethodSendsTheUserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		call func(*geostat.Client) error
	}{
		{"Folder", "folder-labour-en.json", func(c *geostat.Client) error {
			_, err := c.Folder(t.Context(), "en", "Social Statistics/Labour")

			return err
		}},
		{"Table", "wages-region-en.json", func(c *geostat.Client) error {
			_, err := c.Table(t.Context(), "en", "Social Statistics/Labour/Wages_Region.px")

			return err
		}},
		{"Query", "cpi-monthly-query-en.json", func(c *geostat.Client) error {
			_, err := c.Query(t.Context(), "en", monthlyPath, nil)

			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, requests := serve(t, tt.file, http.StatusOK)
			if err := tt.call(client); err != nil {
				t.Fatalf("error = %v", err)
			}

			got := (*requests)[0].header
			agent := got.Get("User-Agent")

			if !strings.HasPrefix(agent, "georgia-mcp/") ||
				!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
				t.Errorf("User-Agent = %q", agent)
			}

			if accept := got.Get("Accept"); accept != "application/json" {
				t.Errorf("Accept = %q, want application/json", accept)
			}
		})
	}
}

// GEOSTAT-SC-002: The client escapes the path and ends it with a slash.
func TestTheClientEndsEachPathWithASlash(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "cpi-monthly-en.json", http.StatusOK)

	metadata, err := client.Table(t.Context(), "en", monthlyPath)
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	want := "/en/Database/Price%20Statistics/CPI/National%20Consumer%20Price%20Index/" +
		"CPI_in_Georgia_%28previous_month=100%29_orenovani2.px/"
	if got := (*requests)[0].rawPath; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}

	if len(metadata.Variables) != 4 || metadata.Variables[3].Code != "month" {
		t.Errorf("Variables = %+v, want 4 ending with month", metadata.Variables)
	}

	rootClient, rootRequests := serve(t, "root-en.json", http.StatusOK)

	nodes, err := rootClient.Folder(t.Context(), "en", "")
	if err != nil {
		t.Fatalf("Folder() error = %v", err)
	}

	if got := (*rootRequests)[0].path; got != "/en/Database/" {
		t.Errorf("path = %q, want /en/Database/", got)
	}

	agriculture := geostat.Node{ID: "Agriculture", Type: "l", Text: "Agriculture"}
	if len(nodes) != 17 || nodes[0] != agriculture {
		t.Errorf("Folder() = %+v", nodes)
	}
}

// GEOSTAT-SC-003: Query sends the selection with the filter item, and reads an answer that
// starts with a byte order mark.
func TestQuerySendsTheSelection(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "cpi-monthly-query-en.json", http.StatusOK)

	dataset, err := client.Query(t.Context(), "en", monthlyPath, []geostat.Selection{
		{Code: "Region", Values: []string{"0"}},
		{Code: "Groups", Values: []string{"0", "1"}},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	sent := (*requests)[0]
	if sent.method != http.MethodPost {
		t.Errorf("method = %s, want POST", sent.method)
	}

	var body any
	if err := json.Unmarshal(sent.body, &body); err != nil {
		t.Fatal(err)
	}

	var want any

	_ = json.Unmarshal([]byte(`{"query":[`+
		`{"code":"Region","selection":{"filter":"item","values":["0"]}},`+
		`{"code":"Groups","selection":{"filter":"item","values":["0","1"]}}],`+
		`"response":{"format":"json"}}`), &want)

	if !reflect.DeepEqual(want, body) {
		t.Errorf("body = %s, want %v", sent.body, want)
	}

	wantData := []geostat.Datum{
		{Key: []string{"0", "0", "22", "9"}, Values: []string{"100.9"}},
		{Key: []string{"0", "1", "22", "9"}, Values: []string{"100.0"}},
	}
	if !reflect.DeepEqual(wantData, dataset.Data) {
		t.Errorf("Data = %+v, want %+v", dataset.Data, wantData)
	}

	if len(dataset.Columns) != 5 || dataset.Columns[4].Type != "c" {
		t.Errorf("Columns = %+v", dataset.Columns)
	}
}

// GEOSTAT-SC-004: A status that is not a success is an *APIError.
func TestAFailureIsAnAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		status int
	}{
		{"the page of IIS", "not-found.html", http.StatusNotFound},
		{"a parameter error", "parameter-error.json", http.StatusNotFound},
		{"a failure", "", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, _ := serve(t, tt.file, tt.status)
			_, err := client.Query(t.Context(), "en", monthlyPath, nil)

			var apiErr *geostat.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status {
				t.Fatalf("Query() error = %v, want an *APIError with %d", err, tt.status)
			}

			if !errors.Is(err, geostat.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false")
			}
		})
	}
}
