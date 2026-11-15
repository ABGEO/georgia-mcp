package napr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the NAPR map.
const DefaultBaseURL = "https://maps.gov.ge"

const coordinateDecimals = 6

var (
	labelPattern = regexp.MustCompile(`^[a-z_]+:[A-Za-z0-9+/]+$`)
	textLine     = regexp.MustCompile(`^(\s*"[a-z_]+"\s*:\s*")(.*)("\s*,?\s*)$`)
)

// Client reads the cadastral map of NAPR. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the NAPR map.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

type searchAnswer struct {
	Result []SearchResult `json:"result"`
}

type nearbyAnswer struct {
	Result []NearbyParcel `json:"result"`
}

type infoAnswer struct {
	Data []ParcelInfo `json:"data"`
}

type shapeAnswer struct {
	Data []Shape `json:"data"`
}

// Search finds a parcel by its cadastral code, or addresses by a text, in the order of
// the map.
func (c *Client) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, ErrEmptyKeyword
	}

	form := url.Values{"keyword": {keyword}}
	body := form.Encode() + "&keyword_description="

	var answer searchAnswer

	request := vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"map", "portal", "search"},
		Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:   []byte(body),
	}
	if err := c.http.FetchJSON(ctx, request, &answer); err != nil {
		return nil, fmt.Errorf("searching the map: %w", err)
	}

	return answer.Result, nil
}

// ParcelsAt returns the parcels near a point, nearest first.
func (c *Client) ParcelsAt(ctx context.Context, lat, lon float64) ([]NearbyParcel, error) {
	request := vendorhttp.Request{
		Path: []string{"lr", "bo", "mg", "getinfo2"},
		Query: url.Values{
			"LON":        {strconv.FormatFloat(lon, 'f', coordinateDecimals, 64)},
			"LAT":        {strconv.FormatFloat(lat, 'f', coordinateDecimals, 64)},
			"C":          {"0"},
			"FRAME_NAME": {"BY_COORDS_AND_COMPASS.GETINFO.MAPGOV"},
		},
	}

	body, err := c.http.Fetch(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("reading the parcels at a point: %w", err)
	}

	var answer nearbyAnswer
	if err := json.Unmarshal(repair(body), &answer); err != nil {
		return nil, fmt.Errorf("decoding the parcels at a point: %w", err)
	}

	return answer.Result, nil
}

// Parcel returns the detail of the parcel with a label: "lr_parcels:AAArWyAATAAIs9BAAB".
func (c *Client) Parcel(ctx context.Context, label string) ([]ParcelInfo, error) {
	var answer infoAnswer
	if err := c.fetchLabel(ctx, parcelPath(), label, "&fmt=json&lang=ka", &answer); err != nil {
		return nil, fmt.Errorf("reading the parcel: %w", err)
	}

	return answer.Data, nil
}

// ParcelShape returns the shape of the parcel with a label.
func (c *Client) ParcelShape(ctx context.Context, label string) ([]Shape, error) {
	var answer shapeAnswer
	if err := c.fetchLabel(ctx, parcelPath(), label, "&res=shp", &answer); err != nil {
		return nil, fmt.Errorf("reading the shape of the parcel: %w", err)
	}

	return answer.Data, nil
}

// AddressShape returns the shape of the address with a label:
// "ract_g_borders:AAAlnvAAyAAEGARAAD".
func (c *Client) AddressShape(ctx context.Context, label string) ([]Shape, error) {
	var answer shapeAnswer

	path := []string{"ar", "mg", "getinfo"}
	if err := c.fetchLabel(ctx, path, label, "&res=shp", &answer); err != nil {
		return nil, fmt.Errorf("reading the shape of the address: %w", err)
	}

	return answer.Data, nil
}

// Label returns the value of lbl in a link of the map, and false when the link has none
// or the value is not a label.
func Label(link string) (string, bool) {
	_, query, found := strings.Cut(link, "?")
	if !found {
		return "", false
	}

	for part := range strings.SplitSeq(query, "&") {
		if value, ok := strings.CutPrefix(part, "lbl="); ok && labelPattern.MatchString(value) {
			return value, true
		}
	}

	return "", false
}

func parcelPath() []string {
	return []string{"lr", "bo", "mg", "getinfo.alpha"}
}

// fetchLabel sends the label with no encoding: the map answers no data for an encoded one.
func (c *Client) fetchLabel(
	ctx context.Context,
	path []string,
	label, rest string,
	target any,
) error {
	if !labelPattern.MatchString(label) {
		return fmt.Errorf("%w: %q", ErrInvalidLabel, label)
	}

	request := vendorhttp.Request{Path: path, RawQuery: "lbl=" + label + rest}
	if err := c.http.FetchJSON(ctx, request, target); err != nil {
		return fmt.Errorf("fetching %s: %w", label, err)
	}

	return nil
}

// repair escapes each quotation mark inside a text value. The map writes one value on
// each line and does not escape a quotation mark in an address.
func repair(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))

	for i, line := range lines {
		match := textLine.FindSubmatchIndex(line)
		if match == nil {
			continue
		}

		value := line[match[4]:match[5]]
		if !bytes.Contains(value, []byte(`"`)) {
			continue
		}

		fixed := slices.Concat(line[:match[4]], escapeQuotes(value), line[match[5]:])
		lines[i] = fixed
	}

	return bytes.Join(lines, []byte("\n"))
}

func escapeQuotes(value []byte) []byte {
	out := make([]byte, 0, len(value)+4) //nolint:mnd // Room for two escaped pairs.

	for i, b := range value {
		if b == '"' && (i == 0 || value[i-1] != '\\') {
			out = append(out, '\\')
		}

		out = append(out, b)
	}

	return out
}
