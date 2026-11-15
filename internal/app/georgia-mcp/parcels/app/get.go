package app

import (
	"context"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
)

// GetParcel returns one parcel by its cadastral code.
type GetParcel struct {
	source ParcelSource
}

// NewGetParcel returns the use case over the source.
func NewGetParcel(source ParcelSource) *GetParcel {
	return &GetParcel{source: source}
}

// Get returns the parcel with the code.
func (u *GetParcel) Get(ctx context.Context, code domain.CadastralCode) (ParcelRead, error) {
	read, err := u.source.Parcel(ctx, code)
	if err != nil {
		return ParcelRead{}, fmt.Errorf("reading the parcel %s: %w", code, err)
	}

	return read, nil
}
