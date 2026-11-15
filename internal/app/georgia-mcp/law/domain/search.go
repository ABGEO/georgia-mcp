package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// DocumentType is the class of an act, as a code of this server: "law".
type DocumentType string

// The document types, in the order of DocumentTypes.
const (
	TypeLaw                         DocumentType = "law"
	TypeParliamentResolution        DocumentType = "parliament_resolution"
	TypePresidentialDecree          DocumentType = "presidential_decree"
	TypeGovernmentResolution        DocumentType = "government_resolution"
	TypeGovernmentDecree            DocumentType = "government_decree"
	TypeEdict                       DocumentType = "edict"
	TypeOrder                       DocumentType = "order"
	TypeInternationalTreaty         DocumentType = "international_treaty"
	TypeConstitutionalCourtDecision DocumentType = "constitutional_court_decision"
	TypeCourtAct                    DocumentType = "court_act"
	TypeSelfGovernmentOrdinance     DocumentType = "self_government_ordinance"
	TypeSelfGovernmentDecision      DocumentType = "self_government_decision"
	TypeRevenueServiceOrder         DocumentType = "revenue_service_order"
	TypeIndividualLegalAct          DocumentType = "individual_legal_act"
	TypeEULaw                       DocumentType = "eu_law"
	TypeDraft                       DocumentType = "draft"
)

// DocumentTypes returns every document type.
func DocumentTypes() []DocumentType {
	return []DocumentType{
		TypeLaw, TypeParliamentResolution, TypePresidentialDecree, TypeGovernmentResolution,
		TypeGovernmentDecree, TypeEdict, TypeOrder, TypeInternationalTreaty,
		TypeConstitutionalCourtDecision, TypeCourtAct, TypeSelfGovernmentOrdinance,
		TypeSelfGovernmentDecision, TypeRevenueServiceOrder, TypeIndividualLegalAct,
		TypeEULaw, TypeDraft,
	}
}

// ParseDocumentType returns the document type of a code, with no regard to case.
func ParseDocumentType(text string) (DocumentType, error) {
	code := DocumentType(strings.ToLower(strings.TrimSpace(text)))
	if !slices.Contains(DocumentTypes(), code) {
		return "", fmt.Errorf("%w: %q", ErrUnknownDocumentType, text)
	}

	return code, nil
}

// ActStatus is the status of an act. The empty status is unknown.
type ActStatus string

const (
	// StatusActive is an act in force.
	StatusActive ActStatus = "active"
	// StatusExpired is an act no longer in force.
	StatusExpired ActStatus = "expired"
	// StatusPending is an act published and not yet in force.
	StatusPending ActStatus = "pending"
)

// ParseActStatus returns the status of a text, with no regard to case.
func ParseActStatus(text string) (ActStatus, error) {
	status := ActStatus(strings.ToLower(strings.TrimSpace(text)))

	switch status {
	case StatusActive, StatusExpired, StatusPending:
		return status, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownActStatus, text)
	}
}

// Order is the order of the results of a search, by the signing date.
type Order int

const (
	// NewestFirst puts the newest signing date first.
	NewestFirst Order = iota
	// OldestFirst puts the oldest signing date first.
	OldestFirst
)

// SearchQuery is a search of the acts.
type SearchQuery struct {
	// Text is the text to find.
	Text string
	// InText searches the text of the acts, not the titles.
	InText bool
	Type   *DocumentType
	Status *ActStatus
	Issuer string
	Number string
	// From and To bound the signing date, both included.
	From, To *kernel.Date
	Order    Order
	// Page is the page of 20 results, from 1.
	Page     int
	Language kernel.Language
}

// HasFilter tells whether the query has a type, a status, an issuer, a number, or a date.
func (q SearchQuery) HasFilter() bool {
	return q.Type != nil || q.Status != nil || q.Issuer != "" || q.Number != "" ||
		q.From != nil || q.To != nil
}

// ActSummary is one act of a search.
type ActSummary struct {
	ID     ActID
	Title  string
	Type   string
	Issuer string
	Number string
	// Signed is the zero date when the source gives none.
	Signed kernel.Date
	Status ActStatus
}
