package tas_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tasadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/tas"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	tasclient "github.com/abgeo/georgia-mcp/pkg/tas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata   = "../../../../../../pkg/tas/testdata/"
	approvedNo = "AR11168188"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// bodies keeps the bodies that the server received.
type bodies struct {
	mu   sync.Mutex
	sent []string
}

func (b *bodies) last() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.sent[len(b.sent)-1]
}

// serve answers each call with the file, or with a body that is not a reply when the file
// is empty.
func serve(t *testing.T, file string) (*tasadapter.Adapter, *bodies) {
	t.Helper()

	reply := []byte("not a reply")

	if file != "" {
		read, err := os.ReadFile(testdata + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		reply = read
	}

	received := &bodies{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		received.mu.Lock()
		received.sent = append(received.sent, string(body))
		received.mu.Unlock()

		_, _ = w.Write(reply)
	}))
	t.Cleanup(server.Close)

	client := tasclient.New(vendorhttp.WithBaseURL(server.URL))

	return tasadapter.NewAdapter(client, clock), received
}

func number(t *testing.T, text string) domain.ApplicationNumber {
	t.Helper()

	parsed, err := domain.ParseApplicationNumber(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// TASPERM-SC-009: The recorded application converts to the domain with its page, its
// dates, its status, its kind, and its decision as plain text.
func TestApplication(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "search-number.js")

	read, err := adapter.Application(t.Context(), number(t, approvedNo))
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	got := read.Application
	if got.Number.String() != approvedNo ||
		got.URL != "https://tas.ge/?p=publicpage&documentId=1168188" ||
		got.Registered.String() != "2026-11-04" || got.Deadline == nil ||
		got.Status != domain.StatusApproved {
		t.Errorf("Application = %+v", got)
	}

	if !read.ReadAt.Equal(clock()) ||
		read.ReadAt.Location().String() != kernel.TbilisiZone().String() {
		t.Errorf("ReadAt = %v, want %v in Tbilisi", read.ReadAt, clock())
	}
}

// TASPERM-SC-009: The recorded application keeps its kind, its executor, and its decision
// as plain text, and holds nothing of the applicant.
func TestApplicationKind(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "search-number.js")

	read, err := adapter.Application(t.Context(), number(t, approvedNo))
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	got := read.Application
	if got.ParentCategory != "საჯარო ინფორმაციის გამოთხოვა" || got.Class != "" ||
		len(got.Stages) != 0 || got.Executor != "ეკა მხეიძე" {
		t.Errorf("kind = %q, %q, %v, %q", got.ParentCategory, got.Class, got.Stages, got.Executor)
	}

	if got.DecisionText == "" || strings.Contains(got.DecisionText, "<") {
		t.Errorf("DecisionText = %q, want plain text", got.DecisionText)
	}

	if strings.Contains(fmt.Sprintf("%+v", got), "userId") {
		t.Error("the application holds userId")
	}
}

// TASPERM-SC-009: A number that TAS matches only as a prefix, or does not match, is not an
// application.
func TestApplicationNotFound(t *testing.T) {
	t.Parallel()

	for file, asked := range map[string]string{"search-number.js": "AR1116818", "search-empty.js": approvedNo} {
		adapter, _ := serve(t, file)

		if _, err := adapter.Application(
			t.Context(),
			number(t, asked),
		); !errors.Is(
			err,
			app.ErrApplicationNotFound,
		) {
			t.Errorf(
				"Application(%s) with %s error = %v, want ErrApplicationNotFound",
				asked,
				file,
				err,
			)
		}
	}
}

// TASPERM-SC-009: A match by code keeps the total and drops the decision texts.
func TestMatchByCode(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "search-code.js")

	code, err := domain.ParseCadastralCode("01.18.04.020.005")
	if err != nil {
		t.Fatal(err)
	}

	read, err := adapter.Match(t.Context(), app.SearchFilter{Code: code})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}

	if read.Total != 5 || len(read.Applications) != 5 {
		t.Fatalf("Match() = %d of %d, want 5 of 5", len(read.Applications), read.Total)
	}

	for _, application := range read.Applications {
		if application.DecisionText != "" || application.Registered.IsZero() {
			t.Errorf("application %s has a decision text or no date", application.Number)
		}
	}
}

// TASPERM-SC-009: A period goes as midnight of its first day and the last millisecond of
// its last day, in Tbilisi.
func TestMatchByPeriod(t *testing.T) {
	t.Parallel()

	adapter, received := serve(t, "search-month.js")

	from, _ := kernel.ParseDate("2026-11-04")

	period, err := domain.NewPeriod(from, from)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := adapter.Match(t.Context(), app.SearchFilter{Period: period}); err != nil {
		t.Fatalf("Match() error = %v", err)
	}

	body := received.last()
	for _, want := range []string{"=Date:1793736000000\n", "=Date:1793822399999\n", "=number:500\n"} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not hold %q:\n%s", want, body)
		}
	}
}

// TASPERM-SC-011: A reply that is not a reply fails as an outage of TAS.
func TestMatchFails(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "")

	code, _ := domain.ParseCadastralCode("01.18.04.020.005")

	if _, err := adapter.Match(
		t.Context(),
		app.SearchFilter{Code: code},
	); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Match() error = %v, want ErrSourceUnavailable", err)
	}
}

// TASPERM-SC-011: Each code of the page of TAS gives its status. A draft fails.
func TestStatusOf(t *testing.T) {
	t.Parallel()

	want := map[int]domain.Status{
		1: domain.StatusApproved, 2: domain.StatusRefused, 7: domain.StatusDeficient,
		12: domain.StatusConditional, 11: domain.StatusNotReviewed, 27: domain.StatusRevoked,
		28: domain.StatusCompleted, 25: domain.StatusAwaitingPayment, 5: domain.StatusInReview,
		6: domain.StatusInReview, 9: domain.StatusInReview, 20: domain.StatusInReview,
		21: domain.StatusInReview, 26: domain.StatusInReview,
	}

	for id, status := range want {
		if got, err := tasadapter.StatusOf(id); err != nil || got != status {
			t.Errorf("StatusOf(%d) = %q, %v, want %q", id, got, err, status)
		}
	}

	if _, err := tasadapter.StatusOf(8); !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("StatusOf(8) error = %v, want ErrSourceUnavailable", err)
	}
}

// TASPERM-SC-010: The HTML reader keeps the words and the lines of the blocks.
func TestPlainText(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		`<p>a&nbsp;b</p><p>&nbsp;</p><p>c<br/>d</p>`: "a b\nc\nd",
		`<p><span>x</span>  <b>y</b></p>`:            "x y",
		``:                                           "",
		`plain`:                                      "plain",
	}

	for in, want := range tests {
		if got := tasadapter.PlainText(in); got != want {
			t.Errorf("PlainText(%q) = %q, want %q", in, got, want)
		}
	}
}

// TASPERM-SC-010: The HTML reader never panics and never leaves a tag.
func FuzzPlainText(f *testing.F) {
	for _, seed := range []string{`<p>a</p>`, `<p`, `&amp;<br>`, `<div><li>x</div>`} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got := tasadapter.PlainText(in)
		if strings.Contains(got, "\n\n") {
			t.Errorf("PlainText(%q) = %q, holds an empty line", in, got)
		}
	})
}

// TASPERM-SC-010: A class keeps its first word. A class of quotation marks is absent.
func TestClassOf(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{"IV კლასი": "IV", `" "`: "", "": "", "-": ""} {
		if got := tasadapter.ClassOf(in); got != want {
			t.Errorf("ClassOf(%q) = %q, want %q", in, got, want)
		}
	}
}
