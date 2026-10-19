package app_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	wissol = "wissol"
	beta   = "beta"
)

// fakeSource answers one product of each grade it sells, after an optional wait.
type fakeSource struct {
	company domain.Company
	grades  []domain.Grade
	err     error
	wait    time.Duration
	calls   atomic.Int32
}

func newFake(code domain.CompanyCode, grades ...domain.Grade) *fakeSource {
	return &fakeSource{company: domain.Company{Code: code, Name: string(code)}, grades: grades}
}

func (f *fakeSource) Company() domain.Company {
	return f.company
}

func (f *fakeSource) Prices(ctx context.Context) (app.PriceList, error) {
	f.calls.Add(1)

	if f.wait > 0 {
		select {
		case <-time.After(f.wait):
		case <-ctx.Done():
			return app.PriceList{}, fmt.Errorf("waiting: %w", ctx.Err())
		}
	}

	if f.err != nil {
		return app.PriceList{}, f.err
	}

	effective, _ := kernel.ParseDate("2026-10-08")
	list := app.PriceList{ReadAt: time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)}

	for i, grade := range f.grades {
		price, err := domain.NewProductPrice(
			fmt.Sprintf("%02d", len(f.grades)-i),
			domain.ProductName{English: string(grade)},
			grade,
			decimal.RequireFromString("4.08"),
			decimal.NullDecimal{},
			effective,
		)
		if err != nil {
			return app.PriceList{}, err
		}

		list.Prices = append(list.Prices, price)
	}

	return list, nil
}
