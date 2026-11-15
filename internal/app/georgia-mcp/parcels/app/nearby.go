package app

import (
	"context"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
)

// NearbyQuery asks for the parcels nearest a point.
type NearbyQuery struct {
	Point domain.Point
	Limit int
}

// ListNearby returns the parcels nearest a point.
type ListNearby struct {
	source ParcelSource
}

// NewListNearby returns the use case over the source.
func NewListNearby(source ParcelSource) *ListNearby {
	return &ListNearby{source: source}
}

// List returns the first Limit parcels, nearest first.
func (u *ListNearby) List(ctx context.Context, query NearbyQuery) (NearbyRead, error) {
	read, err := u.source.ParcelsNear(ctx, query.Point)
	if err != nil {
		return NearbyRead{}, fmt.Errorf("reading the parcels near a point: %w", err)
	}

	read.Parcels = read.Parcels[:min(len(read.Parcels), query.Limit)]

	return read, nil
}
