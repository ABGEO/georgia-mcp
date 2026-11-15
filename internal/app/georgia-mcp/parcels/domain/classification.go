package domain

import "strings"

// LandType is the class of the land of a parcel.
type LandType string

// The land types.
const (
	LandAgricultural    LandType = "agricultural"
	LandNonAgricultural LandType = "non_agricultural"
	LandOther           LandType = "other"
)

// OwnershipType is the kind of the ownership of a parcel.
type OwnershipType string

// The ownership types.
const (
	Ownership      OwnershipType = "ownership"
	CoOwnership    OwnershipType = "co_ownership"
	OwnershipOther OwnershipType = "other"
)

// DocumentKind is the kind of a registry document.
type DocumentKind string

// The document kinds.
const (
	RegistryExtract DocumentKind = "registry_extract"
	CadastralPlan   DocumentKind = "cadastral_plan"
	DocumentOther   DocumentKind = "other"
)

// Classified is a code with the text of the source.
type Classified[T ~string] struct {
	Code T
	// Text is the text as NAPR publishes it, with runs of spaces as one.
	Text string
}

// ParseLandType returns the land type of a text of NAPR. An unknown text gives other.
func ParseLandType(text string) Classified[LandType] {
	return classify(text, func(text string) LandType {
		switch text {
		case "სასოფლო-სამეურნეო":
			return LandAgricultural
		case "არასასოფლო სამეურნეო":
			return LandNonAgricultural
		default:
			return LandOther
		}
	})
}

// ParseOwnershipType returns the ownership type of a text of NAPR. An unknown text gives
// other.
func ParseOwnershipType(text string) Classified[OwnershipType] {
	return classify(text, func(text string) OwnershipType {
		switch text {
		case "საკუთრება":
			return Ownership
		case "თანასაკუთრება":
			return CoOwnership
		default:
			return OwnershipOther
		}
	})
}

// ParseDocumentKind returns the document kind of a text of NAPR. An unknown text gives
// other.
func ParseDocumentKind(text string) Classified[DocumentKind] {
	return classify(text, func(text string) DocumentKind {
		switch text {
		case "ამონაწერი საჯარო რეესტრიდან":
			return RegistryExtract
		case "საკადასტრო გეგმა":
			return CadastralPlan
		default:
			return DocumentOther
		}
	})
}

// CollapseSpaces trims a text and writes each run of spaces as one space.
func CollapseSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func classify[T ~string](text string, code func(string) T) Classified[T] {
	text = CollapseSpaces(text)

	return Classified[T]{Code: code(text), Text: text}
}
