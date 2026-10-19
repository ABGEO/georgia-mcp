package domain

import (
	"fmt"
	"slices"
)

// Grade is the class of a product that every fuel company shares.
type Grade string

const (
	// GradeRegular is petrol with a research octane number of 92 or 93.
	GradeRegular Grade = "regular"
	// GradePremium is petrol with a research octane number of 95 or 96.
	GradePremium Grade = "premium"
	// GradeSuper is petrol with a research octane number of 98.
	GradeSuper Grade = "super"
	// GradeSuper100 is petrol with a research octane number of 100.
	GradeSuper100 Grade = "super_100"
	// GradeDiesel is diesel fuel.
	GradeDiesel Grade = "diesel"
	// GradeCNG is compressed natural gas.
	GradeCNG Grade = "cng"
	// GradeOther is a product that the server cannot classify.
	GradeOther Grade = "other"
)

// Unit is the quantity of a product that one price buys.
type Unit string

const (
	// UnitLitre is one litre.
	UnitLitre Unit = "litre"
	// UnitCubicMetre is one cubic metre.
	UnitCubicMetre Unit = "cubic_metre"
)

// Grades returns every grade, from the petrol of the lowest octane number to other.
func Grades() []Grade {
	return []Grade{
		GradeRegular, GradePremium, GradeSuper, GradeSuper100, GradeDiesel, GradeCNG, GradeOther,
	}
}

// ParseGrade returns the grade of a text. It accepts the exact lower case value only.
func ParseGrade(text string) (Grade, error) {
	grade := Grade(text)
	if !slices.Contains(Grades(), grade) {
		return "", fmt.Errorf("%w: %q", ErrInvalidGrade, text)
	}

	return grade, nil
}

// Unit returns the unit of each price of the grade.
func (g Grade) Unit() Unit {
	if g == GradeCNG {
		return UnitCubicMetre
	}

	return UnitLitre
}
