package domain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	maxIDDigits     = 10
	maxNumberDigits = 4
	insertMark      = "^"
)

// ActID is the identifier of an act at the Legislative Herald: "1043717".
type ActID struct {
	id string
}

// ParseActID returns the act identifier of 1 to 10 ASCII digits with no leading zero.
func ParseActID(text string) (ActID, error) {
	id := strings.TrimSpace(text)
	if !isNumber(id, maxIDDigits) {
		return ActID{}, fmt.Errorf("%w: %q", ErrInvalidActID, text)
	}

	return ActID{id: id}, nil
}

// String returns the identifier.
func (a ActID) String() string {
	return a.id
}

// ArticleNumber is the number of an article: a base, and the superscript of an article
// inserted after it. Article 133³ has the base 133 and the insert 3.
type ArticleNumber struct {
	base, insert int
}

// ParseArticleNumber returns the article number of "82", "133^3", or "133³". Each part
// has 1 to 4 digits with no leading zero.
func ParseArticleNumber(text string) (ArticleNumber, error) {
	trimmed := strings.TrimSpace(text)

	base, insert, inserted := strings.Cut(trimmed, insertMark)
	if !inserted {
		base, insert = splitSuperscript(trimmed)
		inserted = insert != ""
	}

	if !isNumber(base, maxNumberDigits) || (inserted && !isNumber(insert, maxNumberDigits)) {
		return ArticleNumber{}, fmt.Errorf("%w: %q", ErrInvalidArticleNumber, text)
	}

	number := ArticleNumber{base: atoi(base)}
	if inserted {
		number.insert = atoi(insert)
	}

	return number, nil
}

// splitSuperscript returns the text before the first superscript digit, and the
// superscript digits after it as ASCII digits. A text with no superscript gives no insert.
// Any other character after the first superscript keeps it in the insert, so the
// insert fails the check of the digits.
func splitSuperscript(text string) (string, string) {
	index := strings.IndexFunc(text, func(r rune) bool {
		_, ok := superscriptDigit(r)

		return ok
	})
	if index < 0 {
		return text, ""
	}

	var insert strings.Builder

	for _, r := range text[index:] {
		digit, ok := superscriptDigit(r)
		if !ok {
			return text[:index], text[index:]
		}

		insert.WriteRune(digit)
	}

	return text[:index], insert.String()
}

// superscriptDigit returns the ASCII digit of a superscript digit.
func superscriptDigit(char rune) (rune, bool) {
	switch {
	case char == '⁰':
		return '0', true
	case char == '¹':
		return '1', true
	case char == '²':
		return '2', true
	case char == '³':
		return '3', true
	case char >= '⁴' && char <= '⁹':
		return '4' + (char - '⁴'), true
	default:
		return 0, false
	}
}

// String returns "82" or "133^3".
func (n ArticleNumber) String() string {
	if n.insert == 0 {
		return strconv.Itoa(n.base)
	}

	return strconv.Itoa(n.base) + insertMark + strconv.Itoa(n.insert)
}

// IsZero tells whether the number is the zero value.
func (n ArticleNumber) IsZero() bool {
	return n.base == 0
}

// isNumber tells whether a text is 1 to maxDigits ASCII digits with no leading zero.
func isNumber(text string, maxDigits int) bool {
	if text == "" || len(text) > maxDigits || text[0] == '0' {
		return false
	}

	return !strings.ContainsFunc(text, func(r rune) bool { return r < '0' || r > '9' })
}

func atoi(text string) int {
	value, _ := strconv.Atoi(text)

	return value
}

// Version is one consolidated text of an act, with its number and its publication date.
type Version struct {
	Number int
	Date   kernel.Date
}

// PartKind is the kind of a part of an act.
type PartKind string

const (
	// KindSection is a section (კარი).
	KindSection PartKind = "section"
	// KindChapter is a chapter (თავი).
	KindChapter PartKind = "chapter"
	// KindArticle is an article (მუხლი).
	KindArticle PartKind = "article"
)

// Heading is one entry of the table of contents.
type Heading struct {
	Kind PartKind
	// Number is "I" for a section or a chapter, "133^3" for an article.
	Number string
	Title  string
}

// Article is one article of an act, with its paragraphs as published.
type Article struct {
	Number     ArticleNumber
	Title      string
	Paragraphs []string
}

// Body holds the parts of an act, or its text when the page marks no part.
type Body struct {
	// Contents holds every heading, in the order of the act.
	Contents []Heading
	// Articles holds every article, in the order of the act.
	Articles []Article
	// Text is the text of an act with no article. It is empty when Articles is not.
	Text string
}

// Act is one version of a legal act.
type Act struct {
	ID               ActID
	Title            string
	Type             string
	Issuer           string
	Number           string
	Signed           kernel.Date
	RegistrationCode string
	Language         kernel.Language
	// Version is the version of this text.
	Version Version
	// Versions holds every version of the act, the newest first.
	Versions []Version
	Body     Body
}

// VersionOn returns the newest version whose date is on or before a date, the higher
// number on a tie. It returns false when no version is that old.
func (a Act) VersionOn(date kernel.Date) (Version, bool) {
	var (
		best  Version
		found bool
	)

	for _, version := range a.Versions {
		if version.Date.After(date) {
			continue
		}

		if !found || version.Date.After(best.Date) ||
			(version.Date == best.Date && version.Number > best.Number) {
			best, found = version, true
		}
	}

	return best, found
}

// HasVersion tells whether the act has a version with a number.
func (a Act) HasVersion(number int) bool {
	for _, version := range a.Versions {
		if version.Number == number {
			return true
		}
	}

	return false
}
