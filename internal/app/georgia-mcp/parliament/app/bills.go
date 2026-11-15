package app

import (
	"context"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
)

// GeorgianOnlyNotice tells the model why a text with no Georgian letter found nothing.
const GeorgianOnlyNotice = "The Parliament publishes Georgian text only. Search in Georgian, " +
	"for example ბიუჯეტი for budget."

// BillResult is one page of bills, with the notice of a text that is not Georgian.
type BillResult struct {
	BillPage

	Notice string
}

// SearchBills returns the bills that match a filter, newest first.
type SearchBills struct {
	source BillSource
}

// NewSearchBills returns the use case over the source.
func NewSearchBills(source BillSource) *SearchBills {
	return &SearchBills{source: source}
}

// Search refuses a filter with no criterion and reads one page.
func (u *SearchBills) Search(
	ctx context.Context,
	filter BillFilter,
	page domain.Page,
) (BillResult, error) {
	if filter.IsZero() {
		return BillResult{}, ErrNoCriterion
	}

	read, err := u.source.SearchBills(ctx, filter, page)
	if err != nil {
		return BillResult{}, fmt.Errorf("searching the bills: %w", err)
	}

	return BillResult{BillPage: read, Notice: notice(read.Total, filter.Text)}, nil
}

// GetBill returns one bill with its stages.
type GetBill struct {
	source BillSource
}

// NewGetBill returns the use case over the source.
func NewGetBill(source BillSource) *GetBill {
	return &GetBill{source: source}
}

// Get reads the bill with the identifier.
func (u *GetBill) Get(ctx context.Context, id domain.ID) (BillRead, error) {
	read, err := u.source.Bill(ctx, id)
	if err != nil {
		return BillRead{}, fmt.Errorf("reading the bill %d: %w", id, err)
	}

	return read, nil
}

// notice returns GeorgianOnlyNotice when nothing matched and a text has no Georgian letter.
func notice(total int, texts ...domain.SearchText) string {
	if total > 0 {
		return ""
	}

	for _, text := range texts {
		if !text.IsZero() && !text.HasGeorgian() {
			return GeorgianOnlyNotice
		}
	}

	return ""
}
