package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	today      = "2026-10-20"
	octoberOne = "2026-10-01"
	amex       = "amex"
	visa       = "visa"
	visaName   = "Visa"
	upperBOG   = "BOG"
	premium    = "premium"
	points     = "points"
)

func mustDate(t *testing.T, text string) kernel.Date {
	t.Helper()

	date, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return date
}

func validOffer() domain.Offer {
	return domain.Offer{
		Bank:       "bog",
		ID:         "17409",
		Title:      "Ronny's Pizza",
		Benefit:    "20% Discount, unlimited",
		Kinds:      []domain.BenefitKind{domain.KindDiscount},
		Categories: []domain.Category{domain.CategoryDining},
		Tiers:      []domain.Tier{domain.TierStandard},
		URL:        "https://bankofgeorgia.ge/en/offers-hub/details/17409",
	}
}

type newOfferCase struct {
	name    string
	change  func(*domain.Offer)
	wantErr error
}

// BOGOFFER-SC-004: NewOffer accepts a valid offer and refuses an offer that lacks a
// tier, a category, a kind, a title, or an https address.
func TestNewOffer(t *testing.T) {
	t.Parallel()

	for _, tt := range newOfferCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			offer := validOffer()
			tt.change(&offer)

			if _, err := domain.NewOffer(offer); !errors.Is(err, tt.wantErr) {
				t.Errorf("NewOffer() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func newOfferCases() []newOfferCase {
	return []newOfferCase{
		{name: "valid", change: func(*domain.Offer) {}},
		{
			name:    "no tier",
			change:  func(o *domain.Offer) { o.Tiers = nil },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "no category",
			change:  func(o *domain.Offer) { o.Categories = nil },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "no kind",
			change:  func(o *domain.Offer) { o.Kinds = nil },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "empty title",
			change:  func(o *domain.Offer) { o.Title = "  " },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "empty id",
			change:  func(o *domain.Offer) { o.ID = "" },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "http address",
			change:  func(o *domain.Offer) { o.URL = "http://bankofgeorgia.ge/en/offers-hub/details/17409" },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "relative address",
			change:  func(o *domain.Offer) { o.URL = "/details/1" },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "bank code",
			change:  func(o *domain.Offer) { o.Bank = upperBOG },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "unknown tier",
			change:  func(o *domain.Offer) { o.Tiers = []domain.Tier{"vip"} },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name:    "card with no name",
			change:  func(o *domain.Offer) { o.Cards = []domain.Card{{Code: amex}} },
			wantErr: domain.ErrInvalidOffer,
		},
		{
			name: "other and cashback",
			change: func(o *domain.Offer) {
				o.Kinds = []domain.BenefitKind{domain.KindOther, domain.KindCashback}
			},
		},
	}
}

// BOGOFFER-SC-004: NewOffer drops other beside a second kind, removes duplicates, and
// sorts each list.
func TestNewOfferNormalizesTheLists(t *testing.T) {
	t.Parallel()

	offer := validOffer()
	offer.Kinds = []domain.BenefitKind{domain.KindOther, domain.KindCashback, domain.KindCashback}
	offer.Tiers = []domain.Tier{domain.TierPrivate, domain.TierPremium, domain.TierPrivate}
	offer.Categories = []domain.Category{domain.CategoryOther, domain.CategoryLeisure}
	offer.Cards = []domain.Card{
		{Code: visa, Name: visaName},
		{Code: amex, Name: "Amex"},
		{Code: visa, Name: visaName},
	}

	got, err := domain.NewOffer(offer)
	if err != nil {
		t.Fatalf("NewOffer() error = %v", err)
	}

	want := offer
	want.Kinds = []domain.BenefitKind{domain.KindCashback}
	want.Tiers = []domain.Tier{domain.TierPremium, domain.TierPrivate}
	want.Categories = []domain.Category{domain.CategoryLeisure, domain.CategoryOther}
	want.Cards = []domain.Card{{Code: amex, Name: "Amex"}, {Code: visa, Name: visaName}}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("NewOffer() mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-005: A period overlaps a range of dates. A zero start is open to the
// past, and a zero end is open to the future.
func TestPeriodOverlaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		start, end, from, to string
		want                 bool
	}{
		{start: "2026-05-29", end: "2027-05-22", from: today, to: today, want: true},
		{start: "2026-10-30", end: "2026-12-31", from: today, to: today, want: false},
		{start: "2026-10-30", end: "2026-12-31", from: octoberOne, to: "2026-10-31", want: true},
		{start: "2026-09-01", end: octoberOne, from: octoberOne, to: "2026-10-31", want: true},
		{start: "", end: "2026-10-21", from: today, to: today, want: true},
		{start: "2026-01-01", end: "", from: "2030-01-01", to: "2030-01-01", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.start+"_"+tt.end+"_"+tt.from+"_"+tt.to, func(t *testing.T) {
			t.Parallel()

			var start, end kernel.Date
			if tt.start != "" {
				start = mustDate(t, tt.start)
			}

			if tt.end != "" {
				end = mustDate(t, tt.end)
			}

			period, err := domain.NewPeriod(start, end)
			if err != nil {
				t.Fatalf("NewPeriod() error = %v", err)
			}

			if got := period.Overlaps(mustDate(t, tt.from), mustDate(t, tt.to)); got != tt.want {
				t.Errorf("Overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BOGOFFER-SC-005: A period refuses an end before its start, and ends before a date
// only when it has an end.
func TestPeriodBounds(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewPeriod(mustDate(t, "2026-10-31"), mustDate(t, octoberOne)); !errors.Is(
		err, domain.ErrInvalidPeriod,
	) {
		t.Errorf("NewPeriod(end before start) error = %v, want ErrInvalidPeriod", err)
	}

	today := mustDate(t, today)

	ended, _ := domain.NewPeriod(kernel.Date{}, mustDate(t, "2026-10-19"))
	open, _ := domain.NewPeriod(mustDate(t, "2026-01-01"), kernel.Date{})
	last, _ := domain.NewPeriod(kernel.Date{}, today)

	if !ended.EndsBefore(today) || open.EndsBefore(today) || last.EndsBefore(today) {
		t.Errorf("EndsBefore = %v, %v, %v; want true, false, false",
			ended.EndsBefore(today), open.EndsBefore(today), last.EndsBefore(today))
	}
}

func parseError[V any](parse func(string) (V, error)) func(string) error {
	return func(text string) error {
		_, err := parse(text)

		return err
	}
}

// BOGOFFER-SC-006: Each parser refuses a text that is not one of its values.
func TestParsersRefuse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parse   func(string) error
		input   string
		wantErr error
	}{
		{
			name:    "tier",
			parse:   parseError(domain.ParseTier),
			input:   "vip",
			wantErr: domain.ErrInvalidTier,
		},
		{
			name:    "category",
			parse:   parseError(domain.ParseCategory),
			input:   "Leisure",
			wantErr: domain.ErrInvalidCategory,
		},
		{
			name:    "kind",
			parse:   parseError(domain.ParseBenefitKind),
			input:   "money",
			wantErr: domain.ErrInvalidBenefitKind,
		},
		{
			name:    "card",
			parse:   parseError(domain.ParseCardCode),
			input:   "Visa Gold",
			wantErr: domain.ErrInvalidCardCode,
		},
		{
			name:    "bank",
			parse:   parseError(domain.ParseBankCode),
			input:   upperBOG,
			wantErr: domain.ErrInvalidBankCode,
		},
		{name: "valid tier", parse: parseError(domain.ParseTier), input: premium},
		{name: "valid category", parse: parseError(domain.ParseCategory), input: "health_beauty"},
		{name: "valid kind", parse: parseError(domain.ParseBenefitKind), input: points},
		{
			name:  "valid card",
			parse: parseError(domain.ParseCardCode),
			input: "mastercard_world_elite",
		},
		{name: "valid bank", parse: parseError(domain.ParseBankCode), input: "liberty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.parse(tt.input); !errors.Is(err, tt.wantErr) {
				t.Errorf("parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// BOGOFFER-SC-006: The lists of values keep the order of requirements 5.1 to 5.3.
func TestValueLists(t *testing.T) {
	t.Parallel()

	if got := len(domain.Categories()); got != 14 {
		t.Errorf("len(Categories()) = %d, want 14", got)
	}

	wantTiers := []domain.Tier{"standard", "premium", "private", "youth"}
	if diff := cmp.Diff(wantTiers, domain.Tiers()); diff != "" {
		t.Errorf("Tiers() mismatch (-want +got):\n%s", diff)
	}

	wantKinds := []domain.BenefitKind{"discount", "cashback", "points", "other"}
	if diff := cmp.Diff(wantKinds, domain.BenefitKinds()); diff != "" {
		t.Errorf("BenefitKinds() mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-006: CardCodeOf turns a code of a bank into a card code.
func TestCardCodeOf(t *testing.T) {
	t.Parallel()

	tests := map[string]domain.CardCode{
		"GOLD_CARD":   "gold_card",
		"Pupil'sCard": "pupil_scard",
		"Top Card":    "top_card",
	}

	for input, want := range tests {
		if got := domain.CardCodeOf(input); got != want {
			t.Errorf("CardCodeOf(%q) = %q, want %q", input, got, want)
		}
	}
}

// BOGOFFER-SC-006: No text panics a parser of a tool input, and each accepted text
// parses back to itself.
func FuzzParsers(f *testing.F) {
	for _, seed := range []string{"bog", upperBOG, "premium", "health_beauty", "points", "visa_infinite", "", "a b", "ა"} {
		f.Add(seed)
	}

	parsers := map[string]func(string) (string, error){
		"ParseBankCode":    asText(domain.ParseBankCode),
		"ParseCardCode":    asText(domain.ParseCardCode),
		"ParseTier":        asText(domain.ParseTier),
		"ParseCategory":    asText(domain.ParseCategory),
		"ParseBenefitKind": asText(domain.ParseBenefitKind),
	}

	f.Fuzz(func(t *testing.T, input string) {
		for name, parse := range parsers {
			if got, err := parse(input); err == nil && got != input {
				t.Fatalf("%s(%q) = %q", name, input, got)
			}
		}

		_ = domain.CardCodeOf(input)
	})
}

func asText[V ~string](parse func(string) (V, error)) func(string) (string, error) {
	return func(text string) (string, error) {
		value, err := parse(text)

		return string(value), err
	}
}
