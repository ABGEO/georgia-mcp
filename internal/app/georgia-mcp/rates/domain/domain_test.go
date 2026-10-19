package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const euro = "EUR"

// NBGRATE-SC-002: A code in any case gives the upper case code.
func TestParseCurrencyCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    domain.CurrencyCode
		wantErr error
	}{
		{input: "usd", want: "USD"},
		{input: euro, want: euro},
		{input: "US", wantErr: domain.ErrInvalidCurrencyCode},
		{input: "U$D", wantErr: domain.ErrInvalidCurrencyCode},
		{input: "USDT", wantErr: domain.ErrInvalidCurrencyCode},
		{input: "", wantErr: domain.ErrInvalidCurrencyCode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseCurrencyCode(tt.input)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf(
					"ParseCurrencyCode(%q) = %q, %v, want %q, %v",
					tt.input,
					got,
					err,
					tt.want,
					tt.wantErr,
				)
			}
		})
	}
}

// NBGRATE-SC-002: No input panics, and every accepted code has three upper case letters.
func FuzzParseCurrencyCode(f *testing.F) {
	for _, seed := range []string{"usd", euro, "US", "ÜSD", "\x00\x00\x00"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseCurrencyCode(input)
		if err != nil {
			return
		}

		if len(code) != 3 {
			t.Fatalf("ParseCurrencyCode(%q) = %q, want three letters", input, code)
		}

		for _, r := range code {
			if r < 'A' || r > 'Z' {
				t.Fatalf("ParseCurrencyCode(%q) = %q, want upper case letters", input, code)
			}
		}
	})
}

// NBGRATE-SC-012: The rate per unit divides the rate by its quantity. A quantity that is
// not a power of ten is not valid.
func TestOfficialRatePerUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		quantity int64
		rate     string
		want     string
		wantErr  error
	}{
		{quantity: 1, rate: "2.5982", want: "2.5982"},
		{quantity: 10, rate: "7.0738", want: "0.70738"},
		{quantity: 100, rate: "1.6455", want: "0.016455"},
		{quantity: 10000, rate: "0.0146", want: "0.00000146"},
		{quantity: 12, rate: "1.0000", wantErr: domain.ErrInvalidQuantity},
		{quantity: 0, rate: "1.0000", wantErr: domain.ErrInvalidQuantity},
	}

	for _, tt := range tests {
		t.Run(tt.rate, func(t *testing.T) {
			t.Parallel()

			rate, err := domain.NewOfficialRate(
				"USD",
				"US Dollar",
				tt.quantity,
				decimal.RequireFromString(tt.rate),
				decimal.Zero,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewOfficialRate() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if got := rate.RatePerUnit(); !got.Equal(decimal.RequireFromString(tt.want)) {
				t.Errorf("RatePerUnit() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRateSetFind(t *testing.T) {
	t.Parallel()

	usd, _ := domain.NewOfficialRate(
		"USD",
		"US Dollar",
		1,
		decimal.RequireFromString("2.5982"),
		decimal.Zero,
	)
	eur, _ := domain.NewOfficialRate(
		euro,
		"Euro",
		1,
		decimal.RequireFromString("2.9274"),
		decimal.Zero,
	)

	set := domain.NewRateSet(
		mustDate(t, "2026-10-17"),
		time.Time{},
		[]domain.OfficialRate{usd, eur},
	)

	if set.Rates[0].Currency != euro {
		t.Errorf("Rates[0] = %s, want EUR: the set sorts by currency", set.Rates[0].Currency)
	}

	if got, ok := set.Find("USD"); !ok || got.Currency != "USD" {
		t.Errorf("Find(USD) = %v, %v", got, ok)
	}

	if _, ok := set.Find("XXX"); ok {
		t.Error("Find(XXX) = true, want false")
	}

	if set.Empty() || !(domain.RateSet{}).Empty() {
		t.Error("Empty gives a wrong answer")
	}
}

func mustDate(t *testing.T, value string) kernel.Date {
	t.Helper()

	date, err := kernel.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}

	return date
}
