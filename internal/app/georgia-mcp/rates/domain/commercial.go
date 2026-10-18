package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// CommercialRate is the price in lari at which a bank buys and sells one unit of a
// currency. Buy is what the bank pays. Sell is what the bank asks.
type CommercialRate struct {
	Currency CurrencyCode
	Buy      decimal.Decimal
	Sell     decimal.Decimal
}

// NewCommercialRate returns a commercial rate. Both prices must be more than zero.
func NewCommercialRate(code CurrencyCode, buy, sell decimal.Decimal) (CommercialRate, error) {
	if !buy.IsPositive() || !sell.IsPositive() {
		return CommercialRate{}, fmt.Errorf(
			"%w: %s buy %s sell %s", ErrInvalidCommercialRate, code, buy, sell,
		)
	}

	return CommercialRate{Currency: code, Buy: buy, Sell: sell}, nil
}
