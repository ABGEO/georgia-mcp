package domain_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
)

const tbc = "tbc"

// TBCRATE-SC-005: A code of 2 to 16 lower case letters is a bank code. Any other text is not.
func TestParseBankCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    domain.BankCode
		wantErr error
	}{
		{input: tbc, want: tbc},
		{input: "TBC", wantErr: domain.ErrInvalidBankCode},
		{input: "t", wantErr: domain.ErrInvalidBankCode},
		{input: "tb-c", wantErr: domain.ErrInvalidBankCode},
		{input: "abcdefghijklmnopq", wantErr: domain.ErrInvalidBankCode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseBankCode(tt.input)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf(
					"ParseBankCode(%q) = %q, %v, want %q, %v",
					tt.input, got, err, tt.want, tt.wantErr,
				)
			}
		})
	}
}

// TBCRATE-SC-005: No input panics, and every accepted code has lower case letters only.
func FuzzParseBankCode(f *testing.F) {
	for _, seed := range []string{tbc, "TBC", "t", "tb-c", "ტბც"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseBankCode(input)
		if err != nil {
			return
		}

		for _, r := range code {
			if r < 'a' || r > 'z' {
				t.Fatalf("ParseBankCode(%q) = %q, want lower case letters", input, code)
			}
		}
	})
}

// TBCRATE-SC-006: A commercial rate needs a buy and a sell price more than zero.
func TestNewCommercialRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buy, sell string
		wantErr   error
	}{
		{name: "both positive", buy: "2.5800", sell: "2.6200"},
		{name: "zero buy", buy: "0", sell: "2.6200", wantErr: domain.ErrInvalidCommercialRate},
		{
			name:    "negative sell",
			buy:     "2.5800",
			sell:    "-1",
			wantErr: domain.ErrInvalidCommercialRate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buy, sell := decimal.RequireFromString(tt.buy), decimal.RequireFromString(tt.sell)

			got, err := domain.NewCommercialRate("USD", buy, sell)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCommercialRate() error = %v, want %v", err, tt.wantErr)
			}

			if err == nil && (!got.Buy.Equal(buy) || !got.Sell.Equal(sell)) {
				t.Errorf("NewCommercialRate() = %+v, want buy %s sell %s", got, buy, sell)
			}
		})
	}
}
