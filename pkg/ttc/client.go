package ttc

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of the TTC.
	DefaultBaseURL = "https://transit.ttc.com.ge/pis-gateway/api"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/ttc"
	// maxBodyBytes bounds the read of an answer. The list of every stop is 325 KB.
	maxBodyBytes = 4 << 20
	leaveNow     = "leaveNow"
	localeKey    = "locale"
	apiKeyHeader = "X-Api-Key" //nolint:gosec // The name of a header, not a key.
)

// Client calls the API of the TTC. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// WithAPIKey sets the key that the client sends in the header X-api-key. An empty key
// sends no header.
func WithAPIKey(apiKey string) Option {
	if apiKey == "" {
		return func(*vendorhttp.Client) {}
	}

	return vendorhttp.WithHeader(apiKeyHeader, apiKey)
}

// New returns a client of the API of the TTC.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Stops returns every stop of the TTC, with the names in the locale: ka or en.
func (c *Client) Stops(ctx context.Context, locale string) ([]Stop, error) {
	var stops []Stop
	if err := c.get(ctx, &stops, url.Values{localeKey: {locale}}, "v2", "stops"); err != nil {
		return nil, fmt.Errorf("getting the stops: %w", err)
	}

	return stops, nil
}

// Routes returns every route of the TTC: bus, metro, and cable car.
func (c *Client) Routes(ctx context.Context, locale string) ([]Route, error) {
	query := url.Values{"modes": {"BUS,SUBWAY,GONDOLA"}, localeKey: {locale}}

	var routes []Route
	if err := c.get(ctx, &routes, query, "v3", "routes"); err != nil {
		return nil, fmt.Errorf("getting the routes: %w", err)
	}

	return routes, nil
}

// Route returns a route with its patterns.
func (c *Client) Route(ctx context.Context, id, locale string) (RouteDetail, error) {
	var route RouteDetail
	if err := c.get(ctx, &route, url.Values{localeKey: {locale}}, "v3", "routes", id); err != nil {
		return RouteDetail{}, fmt.Errorf("getting the route %s: %w", id, err)
	}

	return route, nil
}

// PatternStops returns the stops of one pattern of a route, in the order of travel.
func (c *Client) PatternStops(ctx context.Context, id, pattern, locale string) ([]Stop, error) {
	query := url.Values{"patternSuffix": {pattern}, localeKey: {locale}}

	var stops []Stop
	if err := c.get(ctx, &stops, query, "v3", "routes", id, "stops"); err != nil {
		return nil, fmt.Errorf("getting the stops of the route %s: %w", id, err)
	}

	return stops, nil
}

// Schedule returns the timetable of one pattern of a route for the current week.
func (c *Client) Schedule(
	ctx context.Context,
	id, pattern, locale string,
) ([]ScheduleGroup, error) {
	query := url.Values{"patternSuffix": {pattern}, localeKey: {locale}}

	var groups []ScheduleGroup
	if err := c.get(ctx, &groups, query, "v3", "routes", id, "schedule"); err != nil {
		return nil, fmt.Errorf("getting the timetable of the route %s: %w", id, err)
	}

	return groups, nil
}

// ArrivalTimes returns the next arrivals at a stop.
func (c *Client) ArrivalTimes(
	ctx context.Context,
	stopID string,
	arrivalQuery ArrivalTimesQuery,
) ([]ArrivalTime, error) {
	query := url.Values{
		localeKey:                     {arrivalQuery.Locale},
		"ignoreScheduledArrivalTimes": {"false"},
	}
	if arrivalQuery.RouteID != "" {
		query.Set("routeId", arrivalQuery.RouteID)
	}

	if arrivalQuery.Max > 0 {
		query.Set("maxNumberOfArrivalTimes", strconv.Itoa(arrivalQuery.Max))
	}

	var arrivals []ArrivalTime
	if err := c.get(ctx, &arrivals, query, "v2", "stops", stopID, "arrival-times"); err != nil {
		return nil, fmt.Errorf("getting the arrivals at the stop %s: %w", stopID, err)
	}

	return arrivals, nil
}

// Plan returns the itineraries between two points.
func (c *Client) Plan(ctx context.Context, planQuery PlanQuery) (Plan, error) {
	query := url.Values{
		"fromPlace":  {coordinates(planQuery.FromLat, planQuery.FromLon)},
		"toPlace":    {coordinates(planQuery.ToLat, planQuery.ToLon)},
		"departMode": {planQuery.DepartMode},
		"modes":      {strings.Join(planQuery.Modes, ",")},
		"optimize":   {planQuery.Optimize},
		localeKey:    {planQuery.Locale},
	}
	if planQuery.DepartMode != leaveNow {
		query.Set("date", planQuery.Date)
		query.Set("time", planQuery.Time)
	}

	var plan Plan
	if err := c.get(ctx, &plan, query, "v2", "plan"); err != nil {
		return Plan{}, fmt.Errorf("getting the trip plan: %w", err)
	}

	return plan, nil
}

// Geocode returns the places that match a text.
func (c *Client) Geocode(ctx context.Context, q GeocodeQuery) (FeatureCollection, error) {
	query := url.Values{"query": {q.Query}, localeKey: {q.Locale}, "bbox": {q.BBox}}

	var places FeatureCollection
	if err := c.get(ctx, &places, query, "v2", "geocode"); err != nil {
		return FeatureCollection{}, fmt.Errorf("searching the places: %w", err)
	}

	return places, nil
}

// get decodes the body of a 200 into target. Any other status is an *APIError. Each
// segment of the path is escaped.
func (c *Client) get(ctx context.Context, target any, query url.Values, segments ...string) error {
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = url.PathEscape(segment)
	}

	return c.http.FetchJSON(ctx, vendorhttp.Request{Path: escaped, Query: query}, target)
}

func coordinates(lat, lon float64) string {
	return strconv.FormatFloat(lat, 'f', -1, 64) + "," + strconv.FormatFloat(lon, 'f', -1, 64)
}
