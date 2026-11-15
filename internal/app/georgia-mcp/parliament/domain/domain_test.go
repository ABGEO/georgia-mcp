package domain_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	budget       = "ბიუჯეტ"
	dream        = "ფრაქცია „ქართული ოცნება“"
	forGeorgia   = "ფრაქცია „საქართველოსთვის“"
	recordedVote = "../../../../../pkg/parliament/testdata/votings-31507.json"
)

func date(t *testing.T, text string) *kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return &parsed
}

// PARL-SC-004: A text of 1 to 128 characters passes, trimmed. An empty text and a text of
// 129 characters fail.
func TestNewSearchText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text string
		want string
	}{
		{text: " ბიუჯეტ ", want: budget},
		{text: "a", want: "a"},
		{
			text: strings.Repeat("ბ", domain.MaxTextLength),
			want: strings.Repeat("ბ", domain.MaxTextLength),
		},
	}

	for _, tt := range tests {
		got, err := domain.NewSearchText(tt.text)
		if err != nil || got.String() != tt.want || got.IsZero() {
			t.Errorf("NewSearchText(%q) = %q, %v, want %q", tt.text, got, err, tt.want)
		}
	}

	for _, text := range []string{"", "   ", strings.Repeat("ბ", domain.MaxTextLength+1)} {
		if _, err := domain.NewSearchText(text); !errors.Is(err, domain.ErrInvalidText) {
			t.Errorf("NewSearchText(%q) error = %v, want ErrInvalidText", text, err)
		}
	}

	if !(domain.SearchText{}).IsZero() {
		t.Error("the zero SearchText is not zero")
	}
}

// PARL-SC-004: A valid text is never empty, never longer than 128 characters, and gives
// itself again.
func FuzzNewSearchText(f *testing.F) {
	for _, seed := range []string{budget, "", " a ", strings.Repeat("x", 200)} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		parsed, err := domain.NewSearchText(text)
		if err != nil {
			return
		}

		count := utf8.RuneCountInString(parsed.String())
		if count == 0 || count > domain.MaxTextLength {
			t.Errorf("NewSearchText(%q) = %q, %d characters", text, parsed, count)
		}

		again, err := domain.NewSearchText(parsed.String())
		if err != nil || again != parsed {
			t.Errorf("NewSearchText(%q) = %q, and again %q, %v", text, parsed, again, err)
		}
	})
}

// PARL-SC-005: A text with a Georgian letter has Georgian. A Latin text and digits do not.
func TestHasGeorgian(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text string
		want bool
	}{
		{text: budget, want: true},
		{text: "budget ბ", want: true},
		{text: "ᲑᲘᲣᲯᲔᲢᲘ", want: true},
		{text: "Ⴁ", want: true},
		{text: "budget", want: false},
		{text: "2026", want: false},
	}

	for _, tt := range tests {
		text, err := domain.NewSearchText(tt.text)
		if err != nil {
			t.Fatal(err)
		}

		if got := text.HasGeorgian(); got != tt.want {
			t.Errorf("HasGeorgian(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

// PARL-SC-011: An identifier above 0 passes. 0 and a negative number fail.
func TestNewID(t *testing.T) {
	t.Parallel()

	if id, err := domain.NewID(32449); err != nil || id != 32449 {
		t.Errorf("NewID(32449) = %d, %v", id, err)
	}

	for _, value := range []int{0, -1} {
		if _, err := domain.NewID(value); !errors.Is(err, domain.ErrInvalidID) {
			t.Errorf("NewID(%d) error = %v, want ErrInvalidID", value, err)
		}
	}
}

// PARL-SC-011: A period that starts after it ends fails. One end alone passes. No end is
// the zero period.
func TestNewPeriod(t *testing.T) {
	t.Parallel()

	from, to := date(t, "2026-11-12"), date(t, "2026-11-10")

	if _, err := domain.NewPeriod(from, to); !errors.Is(err, domain.ErrInvertedRange) {
		t.Errorf("NewPeriod(12, 10) error = %v, want ErrInvertedRange", err)
	}

	tests := []struct {
		name     string
		from, to *kernel.Date
		zero     bool
	}{
		{name: "both ends", from: to, to: from},
		{name: "the same day", from: from, to: from},
		{name: "a start", from: from},
		{name: "an end", to: to},
		{name: "no end", zero: true},
	}

	for _, tt := range tests {
		period, err := domain.NewPeriod(tt.from, tt.to)
		if err != nil || period.IsZero() != tt.zero || period.From != tt.from ||
			period.To != tt.to {
			t.Errorf("%s: NewPeriod() = %+v, %v", tt.name, period, err)
		}
	}
}

// PARL-SC-012: Page 3 of 20 starts at 40. Page 1 starts at 0.
func TestPageStart(t *testing.T) {
	t.Parallel()

	if got := (domain.Page{Number: 3, Size: 20}).Start(); got != 40 {
		t.Errorf("Start() = %d, want 40", got)
	}

	if got := (domain.Page{Number: 1, Size: 10}).Start(); got != 0 {
		t.Errorf("Start() = %d, want 0", got)
	}
}

// PARL-SC-006: Each name of PARL 5.1 gives its reading. Any other name gives false.
func TestReadingOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  domain.Reading
		found bool
	}{
		{name: "I მოსმენა", want: domain.FirstReading, found: true},
		{name: "II მოსმენა", want: domain.SecondReading, found: true},
		{name: "III მოსმენა", want: domain.ThirdReading, found: true},
		{name: " ერთი მოსმენა ", want: domain.SingleReading, found: true},
		{name: "სხდომაზე განსახილველი პროექტი", want: domain.PlenaryDraft, found: true},
		{name: "ალტერნატიული წინადადება", want: domain.AlternativeProposal, found: true},
		{name: "ინიციირებული ვარიანტი"},
		{name: "საფინანსო-საბიუჯეტო კომიტეტის სხდომაზე განხილული დოკუმენტი"},
		{name: ""},
	}

	for _, tt := range tests {
		got, found := domain.ReadingOf(tt.name)
		if got != tt.want || found != tt.found {
			t.Errorf("ReadingOf(%q) = %q, %v, want %q, %v", tt.name, got, found, tt.want, tt.found)
		}
	}
}

// PARL-SC-006: Each outcome and each choice of PARL-DD-007 gives its code. Any other name
// gives other.
func TestOutcomeAndChoiceOf(t *testing.T) {
	t.Parallel()

	outcomes := map[string]domain.Outcome{
		"საკითხი მიღებულია":          domain.Adopted,
		" საკითხი არ არის მიღებული ": domain.Rejected,
		"foo": domain.OtherOutcome,
		"":    domain.OtherOutcome,
	}

	for name, want := range outcomes {
		if got := domain.OutcomeOf(name); got != want {
			t.Errorf("OutcomeOf(%q) = %q, want %q", name, got, want)
		}
	}

	choices := map[string]domain.Choice{
		"დიახ":       domain.Yes,
		"არა":        domain.No,
		"არ ესწრება": domain.Absent,
		"foo":        domain.OtherChoice,
	}

	for name, want := range choices {
		if got := domain.ChoiceOf(name); got != want {
			t.Errorf("ChoiceOf(%q) = %q, want %q", name, got, want)
		}
	}
}

// PARL-SC-009: A response date gives answered. With none, a deadline before the day gives
// overdue. Any other case gives pending.
func TestQuestionStatusOn(t *testing.T) {
	t.Parallel()

	today := *date(t, "2026-11-15")

	tests := []struct {
		name                string
		responded, deadline *kernel.Date
		want                domain.QuestionStatus
	}{
		{
			name:      "a response",
			responded: date(t, "2026-11-01"),
			deadline:  date(t, "2026-10-01"),
			want:      domain.Answered,
		},
		{name: "a later deadline", deadline: date(t, "2026-11-28"), want: domain.Pending},
		{name: "the deadline today", deadline: date(t, "2026-11-15"), want: domain.Pending},
		{name: "a past deadline", deadline: date(t, "2026-11-14"), want: domain.Overdue},
		{name: "no deadline", want: domain.Pending},
	}

	for _, tt := range tests {
		if got := domain.QuestionStatusOn(tt.responded, tt.deadline, today); got != tt.want {
			t.Errorf("%s: QuestionStatusOn() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// recordedChoices reads the choices of vote 31507 that the client test records.
func recordedChoices(t *testing.T) []domain.MemberChoice {
	t.Helper()

	body, err := os.ReadFile(recordedVote)
	if err != nil {
		t.Fatal(err)
	}

	var answer map[string]json.RawMessage
	if err := json.Unmarshal(body, &answer); err != nil {
		t.Fatal(err)
	}

	var list []map[string]any
	if err := json.Unmarshal(answer["list"], &list); err != nil {
		t.Fatal(err)
	}

	choices := make([]domain.MemberChoice, 0, len(list))

	for _, item := range list {
		faction, _ := item["factionRd"].(string)
		member, _ := item["colleagueRd"].(string)
		result, _ := item["votingResult"].(map[string]any)
		name, _ := result["name"].(string)

		choices = append(choices, domain.MemberChoice{
			Member:  member,
			Faction: faction,
			Choice:  domain.Coded[domain.Choice]{Code: domain.ChoiceOf(name), Name: name},
		})
	}

	return choices
}

// PARL-SC-010: The choices of vote 31507 give the largest faction first and the members of
// no faction last. Each member appears once, in the order of the source.
func TestTally(t *testing.T) {
	t.Parallel()

	choices := recordedChoices(t)
	tallies := domain.Tally(choices)

	type counts struct {
		faction                string
		yes, no, absent, other int
		members                int
	}

	got := make([]counts, 0, len(tallies))
	members := 0

	for _, tally := range tallies {
		got = append(got, counts{
			tally.Faction, tally.Yes, tally.No, tally.Absent, tally.Other, len(tally.Members),
		})
		members += len(tally.Members)
	}

	want := []counts{
		{faction: dream, yes: 70, absent: 8, members: 78},
		{faction: forGeorgia, no: 10, absent: 2, members: 12},
		{faction: "", yes: 9, absent: 2, members: 11},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tally() = %+v, want %+v", got, want)
	}

	if members != len(choices) {
		t.Errorf("members = %d, want %d", members, len(choices))
	}

	if first := tallies[0].Members[0]; first != choices[0] {
		t.Errorf("first member = %+v, want %+v", first, choices[0])
	}
}

// PARL-SC-010: Two factions of the same size come by name. An unknown choice counts as
// other. No choice gives no faction.
func TestTallyOrderAndOther(t *testing.T) {
	t.Parallel()

	member := func(name, faction string, choice domain.Choice) domain.MemberChoice {
		return domain.MemberChoice{
			Member: name, Faction: faction, Choice: domain.Coded[domain.Choice]{Code: choice},
		}
	}

	tallies := domain.Tally([]domain.MemberChoice{
		member("a", "", domain.Yes),
		member("b", "ბ", domain.OtherChoice),
		member("c", "ა", domain.No),
	})

	names := make([]string, 0, len(tallies))
	for _, tally := range tallies {
		names = append(names, tally.Faction)
	}

	if !reflect.DeepEqual(names, []string{"ა", "ბ", ""}) {
		t.Errorf("factions = %q, want ა, ბ, and no faction", names)
	}

	if tallies[1].Other != 1 {
		t.Errorf("other of ბ = %d, want 1", tallies[1].Other)
	}

	if got := domain.Tally(nil); len(got) != 0 {
		t.Errorf("Tally(nil) = %+v, want none", got)
	}
}
