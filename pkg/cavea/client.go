package cavea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Cavea.
	DefaultBaseURL = "https://cavea.ge"

	// MaxAvailabilityIDs is the most session codes that Cavea answers in one call of
	// Availability.
	MaxAvailabilityIDs = 30

	// siteOrigin names the production site with any base address, because Cavea refuses
	// a booking call with no Origin.
	siteOrigin = "https://cavea.ge"

	// maxBodyBytes bounds the read of an answer. The stream of /sessions is about 168 KB.
	maxBodyBytes = 2 << 20

	// maxTicketLength is the longest ticket that the page of Cavea accepts.
	maxTicketLength = 512

	statusError = "error"

	apiSegment      = "api"
	bookingSegment  = "booking"
	sessionsSegment = "sessions"
	moviesKey       = `"movies":`
	sessionsKey     = `"dateGroups":`
)

// Client reads the site of Cavea. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Cavea.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithHeader("Origin", siteOrigin),
		vendorhttp.WithHeader("X-Requested-With", "cavea-web"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// envelope is the shape of every answer of the API.
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Ticket string          `json:"ticket"`
	Seats  json.RawMessage `json:"seats"`
}

// Theaters returns every cinema of Cavea, open or closed, in the order of Cavea.
func (c *Client) Theaters(ctx context.Context) ([]Theater, error) {
	var theaters []Theater
	if err := c.data(
		ctx,
		vendorhttp.Request{Path: []string{apiSegment, "theaters"}},
		&theaters,
	); err != nil {
		return nil, fmt.Errorf("getting the theaters: %w", err)
	}

	return theaters, nil
}

// HallMap returns the label of each hall by its key: "cep_1" gives "IMAX".
func (c *Client) HallMap(ctx context.Context) (map[string]string, error) {
	var halls map[string]string
	if err := c.data(
		ctx,
		vendorhttp.Request{Path: []string{apiSegment, "theaters", "hall-map"}},
		&halls,
	); err != nil {
		return nil, fmt.Errorf("getting the hall map: %w", err)
	}

	return halls, nil
}

// Movies returns the movies now showing, or the movies coming soon, in the order of the
// page.
func (c *Client) Movies(ctx context.Context, comingSoon bool) ([]Movie, error) {
	req := vendorhttp.Request{Path: []string{"movies"}}
	if comingSoon {
		req.Query = url.Values{"filter": {"coming-soon"}}
	}

	var movies []Movie
	if err := c.stream(ctx, req, moviesKey, &movies); err != nil {
		return nil, fmt.Errorf("getting the movies: %w", err)
	}

	return movies, nil
}

// SearchMovies returns the movies whose English name matches text.
func (c *Client) SearchMovies(ctx context.Context, text string) ([]FoundMovie, error) {
	req := vendorhttp.Request{
		Path:  []string{apiSegment, "movies", "search"},
		Query: url.Values{"q": {text}},
	}

	var movies []FoundMovie
	if err := c.data(ctx, req, &movies); err != nil {
		return nil, fmt.Errorf("searching the movies: %w", err)
	}

	return movies, nil
}

// Sessions returns the sessions of every movie, by date, from the page /sessions.
func (c *Client) Sessions(ctx context.Context) ([]DateGroup, error) {
	var groups []DateGroup
	if err := c.stream(
		ctx,
		vendorhttp.Request{Path: []string{sessionsSegment}},
		sessionsKey,
		&groups,
	); err != nil {
		return nil, fmt.Errorf("getting the sessions: %w", err)
	}

	return groups, nil
}

// MovieSessions returns the sessions of one movie, with the price of each class.
func (c *Client) MovieSessions(ctx context.Context, movieID string) ([]MovieSession, error) {
	req := vendorhttp.Request{
		Path: []string{apiSegment, bookingSegment, sessionsSegment, "all", movieID},
	}

	var text string
	if err := c.data(ctx, req, &text); err != nil {
		return nil, fmt.Errorf("getting the sessions of %s: %w", movieID, err)
	}

	var sessions []MovieSession
	if err := json.Unmarshal([]byte(text), &sessions); err != nil {
		return nil, fmt.Errorf(
			"decoding the sessions of %s: %w: %w",
			movieID,
			ErrMalformedData,
			err,
		)
	}

	return sessions, nil
}

// Availability returns the level of each session that Cavea knows: "green", "yellow",
// or another value. It takes at most MaxAvailabilityIDs codes.
func (c *Client) Availability(ctx context.Context, sessionIDs []string) (map[string]string, error) {
	if len(sessionIDs) > MaxAvailabilityIDs {
		return nil, fmt.Errorf("%w: %d codes", ErrTooManySessions, len(sessionIDs))
	}

	req := vendorhttp.Request{
		Path:  []string{apiSegment, sessionsSegment, "availability"},
		Query: url.Values{"ids": {strings.Join(sessionIDs, ",")}},
	}

	var levels map[string]string
	if err := c.data(ctx, req, &levels); err != nil {
		return nil, fmt.Errorf("getting the availability: %w", err)
	}

	return levels, nil
}

// SeatMap returns the seats of the hall of one session. It asks Cavea for a ticket of the
// session, then sends the ticket and the cookies of that answer with the request of the
// seats. It keeps neither.
func (c *Client) SeatMap(ctx context.Context, sessionID string) ([]Seat, error) {
	segment := url.PathEscape(sessionID)

	ticket, cookies, err := c.seatTicket(ctx, segment)
	if err != nil {
		return nil, fmt.Errorf("getting the seat ticket of %s: %w", sessionID, err)
	}

	req := vendorhttp.Request{
		Path:   []string{apiSegment, bookingSegment, "seats", segment},
		Header: http.Header{"X-Seat-Ticket": {ticket}, "Cookie": {cookies}},
	}

	answer, err := c.envelope(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("getting the seats of %s: %w", sessionID, err)
	}

	var seats []Seat
	if err := json.Unmarshal(answer.Seats, &seats); err != nil {
		return nil, fmt.Errorf("decoding the seats of %s: %w: %w", sessionID, ErrMalformedData, err)
	}

	return seats, nil
}

func (c *Client) seatTicket(ctx context.Context, segment string) (string, string, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{apiSegment, bookingSegment, "seat-ticket", segment},
	})
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", c.http.StatusError(resp)
	}

	answer, err := decodeEnvelope(resp.Body)
	if err != nil {
		return "", "", err
	}

	if answer.Ticket == "" || len(answer.Ticket) > maxTicketLength {
		return "", "", ErrNoTicket
	}

	return answer.Ticket, cookieHeader(resp.Header), nil
}

// cookieHeader returns the Cookie header that sends back each cookie of an answer.
func cookieHeader(header http.Header) string {
	pairs := []string{}

	for _, line := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		if err != nil {
			continue
		}

		pairs = append(pairs, cookie.Name+"="+cookie.Value)
	}

	return strings.Join(pairs, "; ")
}

// data decodes the field data of an answer of the API into target.
func (c *Client) data(ctx context.Context, req vendorhttp.Request, target any) error {
	answer, err := c.envelope(ctx, req)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(answer.Data, target); err != nil {
		return fmt.Errorf("%w: %w", ErrMalformedData, err)
	}

	return nil
}

func (c *Client) envelope(ctx context.Context, req vendorhttp.Request) (envelope, error) {
	body, err := c.http.Fetch(ctx, req)
	if err != nil {
		return envelope{}, err
	}

	return decodeEnvelope(body)
}

func decodeEnvelope(body []byte) (envelope, error) {
	var answer envelope
	if err := json.Unmarshal(body, &answer); err != nil {
		return envelope{}, fmt.Errorf("%w: %w", ErrMalformedData, err)
	}

	if answer.Status == statusError {
		return envelope{}, ErrNotSuccess
	}

	return answer, nil
}

// stream reads the data stream of a page, and decodes the JSON value after key into
// target. The decoder stops at the end of the value, so the rest of the stream needs no
// parse.
func (c *Client) stream(ctx context.Context, req vendorhttp.Request, key string, target any) error {
	req.Header = http.Header{"Accept": {"text/x-component"}, "Rsc": {"1"}}

	body, err := c.http.Fetch(ctx, req)
	if err != nil {
		return err
	}

	_, value, found := bytes.Cut(body, []byte(key))
	if !found {
		return ErrNoData
	}

	if err := json.NewDecoder(bytes.NewReader(value)).Decode(target); err != nil {
		return fmt.Errorf("%w: %w", ErrMalformedData, err)
	}

	return nil
}
