package domain

import (
	"fmt"
	"slices"
)

// Tier is a group of customers of a bank that an offer is for.
type Tier string

const (
	// TierStandard is every customer of the bank.
	TierStandard Tier = "standard"
	// TierPremium is the customers of the premium package of the bank.
	TierPremium Tier = "premium"
	// TierPrivate is the customers of the private banking of the bank.
	TierPrivate Tier = "private"
	// TierYouth is the young customers of the bank: pupils and students.
	TierYouth Tier = "youth"
)

// Tiers returns every tier, from every customer to the young customers.
func Tiers() []Tier {
	return []Tier{TierStandard, TierPremium, TierPrivate, TierYouth}
}

// ParseTier returns the tier of a text. It accepts the exact lower case value only.
func ParseTier(text string) (Tier, error) {
	return parseValue(text, Tiers(), ErrInvalidTier)
}

// Category is the class of the partner of an offer that every bank shares.
type Category string

const (
	// CategoryDining is restaurants, cafes, bars, and food delivery.
	CategoryDining Category = "dining"
	// CategoryLeisure is resorts, hotels in Georgia, and the spas of a resort.
	CategoryLeisure Category = "leisure"
	// CategoryTravel is flights, hotels abroad, airports, car rental, and roaming.
	CategoryTravel Category = "travel"
	// CategoryHealthBeauty is clinics, pharmacies, beauty salons, and fitness.
	CategoryHealthBeauty Category = "health_beauty"
	// CategoryAuto is cars, car service, and fuel.
	CategoryAuto Category = "auto"
	// CategoryTransport is public transport and taxis.
	CategoryTransport Category = "transport"
	// CategoryShopping is clothes, accessories, gifts, groceries, and online shops.
	CategoryShopping Category = "shopping"
	// CategoryElectronics is electronics and appliances.
	CategoryElectronics Category = "electronics"
	// CategoryHome is furniture, home goods, and repair.
	CategoryHome Category = "home"
	// CategoryEducation is schools, courses, and trainings.
	CategoryEducation Category = "education"
	// CategoryKids is goods and services for children.
	CategoryKids Category = "kids"
	// CategoryEntertainment is cinemas, events, and games.
	CategoryEntertainment Category = "entertainment"
	// CategoryRealEstate is developers and apartments.
	CategoryRealEstate Category = "real_estate"
	// CategoryOther is a partner that no other category covers.
	CategoryOther Category = "other"
)

// Categories returns every offer category, other last.
func Categories() []Category {
	return []Category{
		CategoryDining,
		CategoryLeisure,
		CategoryTravel,
		CategoryHealthBeauty,
		CategoryAuto,
		CategoryTransport,
		CategoryShopping,
		CategoryElectronics,
		CategoryHome,
		CategoryEducation,
		CategoryKids,
		CategoryEntertainment,
		CategoryRealEstate,
		CategoryOther,
	}
}

// ParseCategory returns the offer category of a text. It accepts the exact lower case
// value only.
func ParseCategory(text string) (Category, error) {
	return parseValue(text, Categories(), ErrInvalidCategory)
}

// BenefitKind is the class of a benefit that the bank states in a field, not in a text.
type BenefitKind string

const (
	// KindDiscount is a lower price at the partner.
	KindDiscount BenefitKind = "discount"
	// KindCashback is a part of the payment that the bank returns as money.
	KindCashback BenefitKind = "cashback"
	// KindPoints is loyalty points that the bank gives for the payment.
	KindPoints BenefitKind = "points"
	// KindOther is a benefit whose kind the bank does not state.
	KindOther BenefitKind = "other"
)

// BenefitKinds returns every benefit kind, other last.
func BenefitKinds() []BenefitKind {
	return []BenefitKind{KindDiscount, KindCashback, KindPoints, KindOther}
}

// ParseBenefitKind returns the benefit kind of a text. It accepts the exact lower case
// value only.
func ParseBenefitKind(text string) (BenefitKind, error) {
	return parseValue(text, BenefitKinds(), ErrInvalidBenefitKind)
}

func parseValue[V ~string](text string, values []V, sentinel error) (V, error) {
	value := V(text)
	if !slices.Contains(values, value) {
		return "", fmt.Errorf("%w: %q", sentinel, text)
	}

	return value, nil
}
