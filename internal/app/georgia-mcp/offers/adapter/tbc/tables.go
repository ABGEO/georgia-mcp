package tbc

import "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"

// The tables map the ids and the codes of TBC Bank, as they were on 2026-10-20.

const (
	groupCategory    = "Category"
	groupProductType = "ProductType"
	groupOfferType   = "OfferType"
	groupCardType    = "CardType"
)

func tierTable() map[string]domain.Tier {
	return map[string]domain.Tier{
		"64mvOqr0aBsBrvFAbtHoz9": domain.TierStandard,
		"5I27EbKG2HykRaU760nvck": domain.TierPremium,
		"1opQpNUWlkhEPLDyVjeGm5": domain.TierYouth,
	}
}

func categoryTable() map[string]domain.Category {
	return map[string]domain.Category{
		"CafeandRestaurant": domain.CategoryDining,
		"Leisure":           domain.CategoryLeisure,
		"Travel":            domain.CategoryTravel,
		"Beauty&Health":     domain.CategoryHealthBeauty,
		"Perfume":           domain.CategoryHealthBeauty,
		"Auto":              domain.CategoryAuto,
		"Transport":         domain.CategoryTransport,
		"Shopping":          domain.CategoryShopping,
		"Clothes":           domain.CategoryShopping,
		"Accessories":       domain.CategoryShopping,
		"OnlinePartners":    domain.CategoryShopping,
		"Groceries":         domain.CategoryShopping,
		"Flowers":           domain.CategoryShopping,
		"Electronics":       domain.CategoryElectronics,
		"Home":              domain.CategoryHome,
		"Education":         domain.CategoryEducation,
		"ForKids":           domain.CategoryKids,
		"Entertainment":     domain.CategoryEntertainment,
		"Developers":        domain.CategoryRealEstate,
		"Other":             domain.CategoryOther,
		"Expats":            domain.CategoryOther,
	}
}

// kindTable holds each offer type. PartnerOffers names who gives the offer, not a kind
// of benefit, so it maps to no kind.
func kindTable() map[string][]domain.BenefitKind {
	return map[string][]domain.BenefitKind{
		"Discount":      {domain.KindDiscount},
		"Cashback":      {domain.KindCashback},
		"PartnerOffers": nil,
	}
}

func cardTable() map[string]domain.Card {
	return map[string]domain.Card{
		groupProductType + ":TBCCard":        {Code: "tbc_card", Name: "TBC Card"},
		groupProductType + ":TBCConceptCard": {Code: "concept_card", Name: "TBC Concept Card"},
		groupProductType + ":CreditCard":     {Code: "credit_card", Name: "Credit Card"},
		groupProductType + ":TravelCard":     {Code: "travel_card", Name: "Travel Card"},
		groupProductType + ":PayLater":       {Code: "pay_later", Name: "Pay Later"},
		groupProductType + ":Installment":    {Code: "installment", Name: "Installment"},
		groupProductType + ":Crypto":         {Code: "crypto", Name: "Crypto"},
		groupProductType + ":Pupil'sCard":    {Code: "pupil_card", Name: "Pupil's Card"},
		groupCardType + ":CreditCard":        {Code: "visa", Name: "Visa"},
		groupCardType + ":MasterCard":        {Code: "mastercard", Name: "Mastercard"},
	}
}
