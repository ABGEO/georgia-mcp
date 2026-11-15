package tas

import (
	"errors"
	"fmt"
	"strings"
)

var errScan = errors.New("not a value of DWR")

// escapeLength is the length of a backslash and the character after it.
const escapeLength = 2

// toJSON converts a JavaScript value of DWR to JSON. A key gets quotation marks,
// new Date(ms) becomes ms, a string in single quotation marks gets double ones, and the
// escape \u0000 goes. Every other token passes as it is.
func toJSON(js string) (string, error) {
	scan := scanner{in: js}
	scan.out.Grow(len(js))

	for scan.pos < len(scan.in) {
		if err := scan.step(); err != nil {
			return "", fmt.Errorf("at %d: %w", scan.pos, err)
		}
	}

	return scan.out.String(), nil
}

type scanner struct {
	in  string
	pos int
	out strings.Builder
}

func (s *scanner) step() error {
	char := s.in[s.pos]

	switch {
	case char == '"' || char == '\'':
		return s.text(char)
	case isLetter(char):
		return s.word()
	default:
		s.out.WriteByte(char)
		s.pos++

		return nil
	}
}

// text copies a string. It keeps each escape but \u0000 and \', and escapes a double
// quotation mark of a string in single ones.
func (s *scanner) text(quote byte) error {
	s.out.WriteByte('"')
	s.pos++

	for s.pos < len(s.in) {
		char := s.in[s.pos]

		switch char {
		case quote:
			s.out.WriteByte('"')
			s.pos++

			return nil
		case '\\':
			if err := s.escape(); err != nil {
				return err
			}
		case '"':
			s.out.WriteString(`\"`)
			s.pos++
		default:
			s.out.WriteByte(char)
			s.pos++
		}
	}

	return fmt.Errorf("%w: open string", errScan)
}

func (s *scanner) escape() error {
	rest := s.in[s.pos:]

	switch {
	case strings.HasPrefix(rest, `\u0000`):
		s.pos += len(`\u0000`)
	case strings.HasPrefix(rest, `\'`):
		s.out.WriteByte('\'')
		s.pos += escapeLength
	case len(rest) >= escapeLength:
		s.out.WriteString(rest[:escapeLength])
		s.pos += escapeLength
	default:
		return fmt.Errorf("%w: open escape", errScan)
	}

	return nil
}

func (s *scanner) word() error {
	start := s.pos
	for s.pos < len(s.in) && (isLetter(s.in[s.pos]) || isDigit(s.in[s.pos])) {
		s.pos++
	}

	word := s.in[start:s.pos]

	switch word {
	case "true", "false", "null":
		s.out.WriteString(word)

		return nil
	case "new":
		return s.date()
	}

	if !strings.HasPrefix(strings.TrimLeft(s.in[s.pos:], " \t\r\n"), ":") {
		return fmt.Errorf("%w: word %q is not a key", errScan, word)
	}

	s.out.WriteString(`"` + word + `"`)

	return nil
}

// date reads " Date(<integer>)" after new.
func (s *scanner) date() error {
	rest := strings.TrimLeft(s.in[s.pos:], " ")
	if !strings.HasPrefix(rest, "Date(") {
		return fmt.Errorf("%w: new of no date", errScan)
	}

	s.pos += len(s.in[s.pos:]) - len(rest) + len("Date(")
	start := s.pos

	if s.pos < len(s.in) && s.in[s.pos] == '-' {
		s.pos++
	}

	for s.pos < len(s.in) && isDigit(s.in[s.pos]) {
		s.pos++
	}

	digits := s.in[start:s.pos]
	if digits == "" || digits == "-" || s.pos >= len(s.in) || s.in[s.pos] != ')' {
		return fmt.Errorf("%w: date of no milliseconds", errScan)
	}

	s.out.WriteString(digits)
	s.pos++

	return nil
}

func isLetter(char byte) bool {
	return char == '_' || char == '$' || (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z')
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}
