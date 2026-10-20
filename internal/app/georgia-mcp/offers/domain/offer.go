package domain

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// Offer is a benefit that a bank gives to its customers at a partner, for a period.
// An offer with no card is for every card of its tiers. Terms is empty when the list
// of the bank holds no terms.
type Offer struct {
	Bank           BankCode
	ID             string
	Title          string
	Partner        string
	Benefit        string
	Kinds          []BenefitKind
	Categories     []Category
	BankCategories []string
	Tiers          []Tier
	Cards          []Card
	Period         Period
	URL            string
	Terms          string
}

// NewOffer returns a valid offer. It removes a duplicate from each list and sorts each
// list: the values in the order of their list function, the cards by code. It drops
// the kind other when the offer has a second kind.
func NewOffer(offer Offer) (Offer, error) {
	for _, check := range []func() error{
		offer.validateRequired,
		offer.validateValues,
		offer.validateCards,
		offer.validateURL,
	} {
		if err := check(); err != nil {
			return Offer{}, fmt.Errorf("checking the offer %q: %w", offer.ID, err)
		}
	}

	offer.Kinds = inOrder(offer.Kinds, BenefitKinds())
	if len(offer.Kinds) > 1 {
		offer.Kinds = slices.DeleteFunc(offer.Kinds, func(kind BenefitKind) bool {
			return kind == KindOther
		})
	}

	offer.Tiers = inOrder(offer.Tiers, Tiers())
	offer.Categories = inOrder(offer.Categories, Categories())

	offer.Cards = slices.SortedFunc(slices.Values(offer.Cards), func(a, b Card) int {
		return cmp.Compare(a.Code, b.Code)
	})
	offer.Cards = slices.CompactFunc(offer.Cards, func(a, b Card) bool {
		return a.Code == b.Code
	})

	return offer, nil
}

func (o Offer) validateRequired() error {
	if _, err := ParseBankCode(string(o.Bank)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidOffer, err)
	}

	switch {
	case o.ID == "":
		return fmt.Errorf("%w: no id", ErrInvalidOffer)
	case strings.TrimSpace(o.Title) == "":
		return fmt.Errorf("%w: no title", ErrInvalidOffer)
	case len(o.Tiers) == 0:
		return fmt.Errorf("%w: no tier", ErrInvalidOffer)
	case len(o.Categories) == 0:
		return fmt.Errorf("%w: no category", ErrInvalidOffer)
	case len(o.Kinds) == 0:
		return fmt.Errorf("%w: no kind", ErrInvalidOffer)
	}

	return nil
}

func (o Offer) validateValues() error {
	return errors.Join(
		validateAll(o.Tiers, ParseTier),
		validateAll(o.Categories, ParseCategory),
		validateAll(o.Kinds, ParseBenefitKind),
	)
}

func (o Offer) validateCards() error {
	for _, card := range o.Cards {
		if _, err := ParseCardCode(string(card.Code)); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidOffer, err)
		}

		if card.Name == "" {
			return fmt.Errorf("%w: the card %q has no name", ErrInvalidOffer, card.Code)
		}
	}

	return nil
}

func (o Offer) validateURL() error {
	address, err := url.Parse(o.URL)
	if err != nil || address.Scheme != "https" || address.Host == "" {
		return fmt.Errorf("%w: the address %q is not absolute https", ErrInvalidOffer, o.URL)
	}

	return nil
}

func validateAll[V ~string](values []V, parse func(string) (V, error)) error {
	for _, value := range values {
		if _, err := parse(string(value)); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidOffer, err)
		}
	}

	return nil
}

// inOrder returns the values with no duplicate, in the order of the full list.
func inOrder[V comparable](values, order []V) []V {
	return slices.DeleteFunc(slices.Clone(order), func(value V) bool {
		return !slices.Contains(values, value)
	})
}
