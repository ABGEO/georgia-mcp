package domain

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	numberPrefix    = "AR1"
	maxNumberDigits = 9
)

// ApplicationNumber is the number of an application: AR1 and 1 to 9 digits.
type ApplicationNumber struct {
	value string
}

// ParseApplicationNumber trims the text and takes upper case. Digits alone get AR1, as the
// page of TAS adds it.
func ParseApplicationNumber(text string) (ApplicationNumber, error) {
	value := strings.ToUpper(strings.TrimSpace(text))
	if value != "" && isDigits(value) {
		value = numberPrefix + value
	}

	digits, found := strings.CutPrefix(value, numberPrefix)
	if !found || digits == "" || len(digits) > maxNumberDigits || !isDigits(digits) {
		return ApplicationNumber{}, fmt.Errorf("%w: %q", ErrInvalidApplicationNumber, text)
	}

	return ApplicationNumber{value: value}, nil
}

// String returns the number: "AR11168188".
func (n ApplicationNumber) String() string {
	return n.value
}

// digits returns the number after AR as a count, for the order of two numbers.
func (n ApplicationNumber) digits() int {
	count, _ := strconv.Atoi(strings.TrimPrefix(n.value, "AR"))

	return count
}
