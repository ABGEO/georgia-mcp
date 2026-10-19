package app

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// The rank of a match: an equal code or number, then a name that starts with the
// text, then a name that holds it.
const (
	rankEqual = iota
	rankPrefix
	rankInside
	noMatch
)

// resolveStop finds the stop of an identifier, or else of a stop code.
func resolveStop(stops []domain.Stop, text string) (domain.Stop, error) {
	text = strings.TrimSpace(text)

	if index := slices.IndexFunc(
		stops,
		func(s domain.Stop) bool { return string(s.ID) == text },
	); index >= 0 {
		return stops[index], nil
	}

	if index := slices.IndexFunc(
		stops,
		func(s domain.Stop) bool { return s.Code != "" && s.Code == text },
	); index >= 0 {
		return stops[index], nil
	}

	return domain.Stop{}, &UnknownStopError{Text: text}
}

// resolveRoute finds the route of an identifier, or else the one route of a number,
// of the kind when the kind is not empty.
func resolveRoute(routes []domain.Route, text string, kind domain.Kind) (domain.Route, error) {
	text = strings.TrimSpace(text)

	if index := slices.IndexFunc(
		routes,
		func(r domain.Route) bool { return string(r.ID) == text },
	); index >= 0 {
		return routes[index], nil
	}

	var found []domain.Route

	for _, route := range routes {
		if route.Number == text && (kind == "" || route.Kind == kind) {
			found = append(found, route)
		}
	}

	switch len(found) {
	case 0:
		return domain.Route{}, &UnknownRouteError{Text: text}
	case 1:
		return found[0], nil
	default:
		return domain.Route{}, &AmbiguousRouteError{Text: text, Routes: found}
	}
}

// classify returns the kind of the route with the number that stops at a stop of the
// kind. An answer of the TTC gives a minibus the mode of a bus, and no route identifier.
func classify(routes []domain.Route, number string, stopKind domain.Kind) domain.Kind {
	for _, route := range routes {
		if route.Number == number && route.Kind.StopKind() == stopKind {
			return route.Kind
		}
	}

	return stopKind
}

// normalize returns the text in lowercase, with each run of spaces as one space.
func normalize(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// rankName returns the rank of a match of a normalized text in a name.
func rankName(name domain.Name, text string) int {
	best := noMatch

	for _, value := range []string{name.Georgian, name.English} {
		value = normalize(value)

		switch {
		case value == "":
		case strings.HasPrefix(value, text):
			best = min(best, rankPrefix)
		case strings.Contains(value, text):
			best = min(best, rankInside)
		}
	}

	return best
}

// ranked is an item of a search with its rank.
type ranked[T any] struct {
	item T
	rank int
}

// sortRanked sorts by rank, then by the name in the language, then by identifier.
func sortRanked[T any](
	items []ranked[T],
	name func(T) domain.Name,
	id func(T) string,
	language kernel.Language,
) []T {
	slices.SortStableFunc(items, func(a, b ranked[T]) int {
		return cmp.Or(
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(name(a.item).In(language), name(b.item).In(language)),
			cmp.Compare(id(a.item), id(b.item)),
		)
	})

	sorted := make([]T, 0, len(items))
	for _, item := range items {
		sorted = append(sorted, item.item)
	}

	return sorted
}

// compareNumbers orders two route numbers as integers, and a text that is not an
// integer after the integers.
func compareNumbers(left, right string) int {
	first, errFirst := strconv.Atoi(left)
	second, errSecond := strconv.Atoi(right)

	switch {
	case errFirst == nil && errSecond == nil:
		return cmp.Compare(first, second)
	case errFirst == nil:
		return -1
	case errSecond == nil:
		return 1
	default:
		return cmp.Compare(left, right)
	}
}

func limit[T any](items []T, count int) []T {
	if count > 0 && len(items) > count {
		return items[:count]
	}

	return items
}
