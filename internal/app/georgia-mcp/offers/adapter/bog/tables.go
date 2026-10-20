package bog

import "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"

// The tables map the codes of Bank of Georgia, as they were on 2026-10-20.

const (
	codeSOLO = "SOLO"
	codePLUS = "PLUS"
)

func tierTable() map[string]domain.Tier {
	return map[string]domain.Tier{
		"RETAIL": domain.TierStandard,
		codeSOLO: domain.TierPremium,
		"WM":     domain.TierPrivate,
	}
}

func kindTable() map[string]domain.BenefitKind {
	return map[string]domain.BenefitKind{
		"DISCOUNT": domain.KindDiscount,
		"CASHBACK": domain.KindCashback,
		codePLUS:   domain.KindPoints,
		"MR":       domain.KindPoints,
		"SCOOLA":   domain.KindOther,
		"OTHER":    domain.KindOther,
	}
}

func categoryTable() map[int]domain.Category {
	return map[int]domain.Category{
		111: domain.CategoryDining,
		108: domain.CategoryLeisure,
		110: domain.CategoryHealthBeauty,
		12:  domain.CategoryTravel,
		113: domain.CategoryAuto,
		109: domain.CategoryShopping,
		5:   domain.CategoryElectronics,
		115: domain.CategoryHome,
		114: domain.CategoryEducation,
		116: domain.CategoryKids,
		112: domain.CategoryEntertainment,
		16:  domain.CategoryOther,
	}
}

func cardTable() map[string]domain.Card {
	return map[string]domain.Card{
		codeSOLO:                 {Code: "solo", Name: codeSOLO},
		codePLUS:                 {Code: "plus", Name: codePLUS},
		"SUPER_PLUS":             {Code: "super_plus", Name: "Super PLUS"},
		"AMEX":                   {Code: "amex", Name: "Amex"},
		"VISA_CARDS":             {Code: "visa", Name: "Visa"},
		"VISA_CREDIT_INFINITE":   {Code: "visa_infinite", Name: "Visa Infinite"},
		"VISA_CREDIT_PLATINUM":   {Code: "visa_platinum", Name: "Visa Platinum"},
		"VISA_CREDIT_SIGNATURE":  {Code: "visa_signature", Name: "Visa Signature"},
		"MASTERCARD":             {Code: "mastercard", Name: "Mastercard"},
		"MASTERCARD_WORLD_ELITE": {Code: "mastercard_world_elite", Name: "Mastercard World Elite"},
		"STUDENTS":               {Code: "student_card", Name: "Student Card"},
		"SCHOOL_CARD":            {Code: "scool_card", Name: "sCool Card"},
		"TEACHER_CARD":           {Code: "teacher_card", Name: "Teacher Card"},
	}
}
