package liberty

import "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"

// The tables map the ids of Liberty, as they were on 2026-10-20.

func categoryTable() map[int]domain.Category {
	return map[int]domain.Category{
		1:  domain.CategoryAuto,
		3:  domain.CategoryDining,
		4:  domain.CategoryShopping,
		5:  domain.CategoryElectronics,
		6:  domain.CategoryLeisure,
		7:  domain.CategoryEducation,
		8:  domain.CategoryKids,
		9:  domain.CategoryShopping,
		10: domain.CategoryHealthBeauty,
		11: domain.CategoryHome,
		13: domain.CategoryOther,
		14: domain.CategoryRealEstate,
	}
}

func kindTable() map[int]domain.BenefitKind {
	return map[int]domain.BenefitKind{
		1: domain.KindDiscount,
		2: domain.KindCashback,
	}
}

func cardTable() map[int]domain.Card {
	return map[int]domain.Card{
		3: {Code: "visa", Name: "Visa"},
		4: {Code: "mastercard", Name: "Mastercard"},
		5: {Code: "top_card_black", Name: "Top|Card Black Edition"},
	}
}
