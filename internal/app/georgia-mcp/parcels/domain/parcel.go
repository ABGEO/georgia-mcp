package domain

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	squareMetres       = "m2"
	sourceSquareMetres = "კვ.მ."
)

var decimal = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// Area is the area of a parcel: a decimal text, and "m2" or the unit of the source.
type Area struct {
	Value string
	Unit  string
}

// NewArea returns the area of a value and a unit of NAPR. "კვ.მ." gives "m2".
func NewArea(value, unit string) (Area, error) {
	value, unit = strings.TrimSpace(value), CollapseSpaces(unit)
	if !decimal.MatchString(value) || unit == "" {
		return Area{}, fmt.Errorf("%w: %q %q", ErrInvalidArea, value, unit)
	}

	if unit == sourceSquareMetres {
		unit = squareMetres
	}

	return Area{Value: value, Unit: unit}, nil
}

// RegistryDocument is one document of the registry about a parcel.
type RegistryDocument struct {
	Kind               Classified[DocumentKind]
	RegistrationNumber string
	// Date is nil when NAPR gives none.
	Date *kernel.Date
	Link string
}

// SortDocuments orders by kind (registry_extract, cadastral_plan, other), then by date,
// newest first, with no date last. It keeps the first of two documents with one link.
func SortDocuments(documents []RegistryDocument) []RegistryDocument {
	seen := make(map[string]bool, len(documents))
	sorted := make([]RegistryDocument, 0, len(documents))

	for _, document := range documents {
		if seen[document.Link] {
			continue
		}

		seen[document.Link] = true
		sorted = append(sorted, document)
	}

	slices.SortStableFunc(sorted, func(first, second RegistryDocument) int {
		if order := cmp.Compare(kindRank(first.Kind.Code), kindRank(second.Kind.Code)); order != 0 {
			return order
		}

		return compareDates(first.Date, second.Date)
	})

	return sorted
}

func kindRank(kind DocumentKind) int {
	switch kind {
	case RegistryExtract:
		return 0
	case CadastralPlan:
		return 1
	case DocumentOther:
		return 2 //nolint:mnd // Last of three kinds.
	default:
		return 2 //nolint:mnd // Last of three kinds.
	}
}

// compareDates puts the newer date first and no date last.
func compareDates(first, second *kernel.Date) int {
	switch {
	case first == nil && second == nil:
		return 0
	case first == nil:
		return 1
	case second == nil:
		return -1
	case first.After(*second):
		return -1
	case first.Before(*second):
		return 1
	default:
		return 0
	}
}

// Parcel is one land parcel of the cadastre.
type Parcel struct {
	Code      CadastralCode
	Address   string
	Area      Area
	LandType  Classified[LandType]
	Ownership Classified[OwnershipType]
	// Owners is empty when NAPR names none.
	Owners []string
	// Documents is in the order of SortDocuments.
	Documents []RegistryDocument
	Centre    Point
}

// NearbyParcel is a parcel near a point.
type NearbyParcel struct {
	Code      CadastralCode
	Address   string
	DistanceM int
}

// Address is an address of the address registry, with its centre point.
type Address struct {
	Text         string
	Municipality string
	Centre       Point
}
