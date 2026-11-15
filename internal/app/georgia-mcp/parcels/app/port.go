package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
)

// AddressRef is the reference of the source to one address. Only the adapter reads it.
type AddressRef string

// AddressMatch is an address that matches a text, before the read of its centre.
type AddressMatch struct {
	Text         string
	Municipality string
	Ref          AddressRef
}

// ParcelRead is a parcel and the instant of its read.
type ParcelRead struct {
	Parcel domain.Parcel
	// MapURL is the page of the parcel on the map of the source.
	MapURL string
	ReadAt time.Time
	Stale  bool
}

// NearbyRead is every parcel that the source gives near a point, nearest first.
type NearbyRead struct {
	Parcels []domain.NearbyParcel
	ReadAt  time.Time
	Stale   bool
}

// MatchRead is the addresses that match a text, in the order of the source.
type MatchRead struct {
	Matches []AddressMatch
	ReadAt  time.Time
	Stale   bool
}

// CentreRead is the centre point of an address.
type CentreRead struct {
	Centre domain.Point
	ReadAt time.Time
	Stale  bool
}

// ParcelSource gives the parcels and the addresses of the cadastre.
type ParcelSource interface {
	// Parcel returns the parcel with the code, or ErrParcelNotFound.
	Parcel(ctx context.Context, code domain.CadastralCode) (ParcelRead, error)
	// ParcelsNear returns the parcels near the point.
	ParcelsNear(ctx context.Context, point domain.Point) (NearbyRead, error)
	// MatchAddresses returns the addresses that match the text.
	MatchAddresses(ctx context.Context, text string) (MatchRead, error)
	// AddressCentre returns the centre point of an address.
	AddressCentre(ctx context.Context, ref AddressRef) (CentreRead, error)
}
