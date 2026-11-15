package tas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the public search of TAS.
const DefaultBaseURL = "https://docs.tbilisi.gov.ge"

// MaxBody holds 500 applications of about 11 KB each, with room.
const MaxBody = 16 << 20

const (
	applicationID   = 2
	callbackPrefix  = `dwr.engine.remote.handleCallback("1","0",`
	scriptSessionID = "00000000000000000000000000000000/georgia-mcp"
)

var (
	codePattern   = regexp.MustCompile(`^[0-9.]+$`)
	numberPattern = regexp.MustCompile(`^[A-Z0-9]+$`)
)

// Client reads the public search of TAS. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the public search of TAS. It reads at most MaxBody bytes
// unless an option sets another limit.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithMaxBody(MaxBody),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

type envelope struct {
	IsSuccess    bool       `json:"isSuccess"`
	ErrorMessage string     `json:"errorMessage"`
	Source       []Document `json:"source"`
	Sources      []int      `json:"sources"`
}

type exception struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// SearchDocuments returns one page of the applications that match the filter, and the
// count of all matches.
func (c *Client) SearchDocuments(ctx context.Context, filter Filter) (Page, error) {
	body, err := requestBody(filter)
	if err != nil {
		return Page{}, err
	}

	request := vendorhttp.Request{
		Method: http.MethodPost,
		Path: []string{
			"architect", "dwr", "call", "plaincall",
			"DocumentManager.getDocsForPublicInfo.dwr",
		},
		Header: http.Header{"Content-Type": {"text/plain"}},
		Body:   []byte(body),
	}

	reply, err := c.http.Fetch(ctx, request)
	if err != nil {
		return Page{}, fmt.Errorf("searching the applications: %w", err)
	}

	page, err := readReply(string(reply))
	if err != nil {
		return Page{}, fmt.Errorf("reading the applications: %w", err)
	}

	return page, nil
}

// requestBody writes the call of DWR. Each value goes into the body with no encoding.
func requestBody(filter Filter) (string, error) {
	if err := validate(filter); err != nil {
		return "", err
	}

	call := callWriter{}
	call.field("applicationId", "number:"+strconv.Itoa(applicationID))
	call.field("start", "number:"+strconv.Itoa(filter.Start))
	call.field("limit", "number:"+strconv.Itoa(filter.Limit))

	if filter.CadastralCode != "" {
		call.field("naprCadCode", "string:"+filter.CadastralCode)
	}

	if filter.DocumentNo != "" {
		call.field("documentNo", "string:"+filter.DocumentNo)
	}

	if !filter.From.IsZero() {
		call.field("fromDate", "Date:"+strconv.FormatInt(filter.From.UnixMilli(), 10))
	}

	if !filter.To.IsZero() {
		call.field("toDate", "Date:"+strconv.FormatInt(filter.To.UnixMilli(), 10))
	}

	if len(filter.StatusIDs) > 0 {
		call.numbers("docStatusIds", filter.StatusIDs)
	}

	return call.String(), nil
}

func validate(filter Filter) error {
	if filter.Limit < 1 || filter.Start < 0 {
		return fmt.Errorf(
			"%w: start %d, limit %d",
			ErrInvalidFilter,
			filter.Start,
			filter.Limit,
		)
	}

	if filter.CadastralCode != "" && !codePattern.MatchString(filter.CadastralCode) {
		return fmt.Errorf("%w: code %q", ErrInvalidFilter, filter.CadastralCode)
	}

	if filter.DocumentNo != "" && !numberPattern.MatchString(filter.DocumentNo) {
		return fmt.Errorf("%w: number %q", ErrInvalidFilter, filter.DocumentNo)
	}

	return nil
}

// callWriter numbers the values of the parameter of a DWR call.
type callWriter struct {
	values []string
	refs   []string
}

func (w *callWriter) String() string {
	lines := slices.Concat(
		[]string{
			"callCount=1",
			"windowName=",
			"nextReverseAjaxIndex=0",
			"c0-scriptName=DocumentManager",
			"c0-methodName=getDocsForPublicInfo",
			"c0-id=0",
		},
		w.values,
		[]string{
			"c0-param0=Object_Object:{" + strings.Join(w.refs, ", ") + "}",
			"batchId=1",
			"instanceId=0",
			"page=%2Farchitect%2FpublicInformation.html",
			"httpSessionId=",
			"scriptSessionId=" + scriptSessionID,
		},
	)

	return strings.Join(lines, "\n") + "\n"
}

func (w *callWriter) next() string {
	return "c0-e" + strconv.Itoa(len(w.values)+1)
}

func (w *callWriter) field(name, value string) {
	id := w.next()
	w.values = append(w.values, id+"="+value)
	w.refs = append(w.refs, name+":reference:"+id)
}

func (w *callWriter) numbers(name string, numbers []int) {
	id := w.next()
	w.values = append(w.values, "")
	at := len(w.values) - 1

	items := make([]string, 0, len(numbers))

	for _, number := range numbers {
		item := w.next()
		w.values = append(w.values, item+"=number:"+strconv.Itoa(number))
		items = append(items, "reference:"+item)
	}

	w.values[at] = id + "=Array:[" + strings.Join(items, ",") + "]"
	w.refs = append(w.refs, name+":reference:"+id)
}

func readReply(reply string) (Page, error) {
	for line := range strings.Lines(reply) {
		if value, found := strings.CutPrefix(strings.TrimSpace(line), callbackPrefix); found {
			return readCallback(value)
		}
	}

	for line := range strings.Lines(reply) {
		if strings.Contains(line, "handleBatchException(") ||
			strings.Contains(line, "handleException(") {
			return Page{}, readException(line)
		}
	}

	return Page{}, fmt.Errorf("%w: no callback", ErrInvalidReply)
}

func readCallback(value string) (Page, error) {
	inside, _, found := strings.CutLast(value, ")")
	if !found {
		return Page{}, fmt.Errorf("%w: open callback", ErrInvalidReply)
	}

	text, err := toJSON(inside)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %w", ErrInvalidReply, err)
	}

	var answer envelope
	if err := json.Unmarshal([]byte(text), &answer); err != nil {
		return Page{}, fmt.Errorf("%w: %w", ErrInvalidReply, err)
	}

	if !answer.IsSuccess {
		return Page{}, fmt.Errorf("%w: %s", ErrRemote, answer.ErrorMessage)
	}

	if len(answer.Sources) == 0 {
		return Page{}, fmt.Errorf("%w: no total", ErrInvalidReply)
	}

	return Page{Total: answer.Sources[0], Documents: answer.Source}, nil
}

func readException(line string) error {
	start, end := strings.Index(line, "{"), strings.LastIndex(line, "}")
	if start < 0 || end < start {
		return fmt.Errorf("%w: an exception with no value", ErrRemote)
	}

	text, err := toJSON(line[start : end+1])
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidReply, err)
	}

	var thrown exception
	if err := json.Unmarshal([]byte(text), &thrown); err != nil {
		return errors.Join(ErrRemote, fmt.Errorf("%w: %w", ErrInvalidReply, err))
	}

	return fmt.Errorf("%w: %s: %s", ErrRemote, thrown.Name, thrown.Message)
}
