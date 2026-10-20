package domain

import "errors"

var (
	// ErrInvalidBankCode tells that a text is not 2 to 16 lower case letters.
	ErrInvalidBankCode = errors.New("bank: invalid code")
	// ErrInvalidTier tells that a text is not one of the tiers.
	ErrInvalidTier = errors.New("tier: invalid")
	// ErrInvalidCategory tells that a text is not one of the offer categories.
	ErrInvalidCategory = errors.New("category: invalid")
	// ErrInvalidBenefitKind tells that a text is not one of the benefit kinds.
	ErrInvalidBenefitKind = errors.New("benefit kind: invalid")
	// ErrInvalidCardCode tells that a text is not 2 to 40 lower case letters, digits,
	// and underscores.
	ErrInvalidCardCode = errors.New("card: invalid code")
	// ErrInvalidPeriod tells that an offer period ends before it starts.
	ErrInvalidPeriod = errors.New("period: invalid")
	// ErrInvalidOffer tells that an offer lacks an id, a title, a tier, a category, a
	// kind, or an https address, or that one of its values is not valid.
	ErrInvalidOffer = errors.New("offer: invalid")
)
