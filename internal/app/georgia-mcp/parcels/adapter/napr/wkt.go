package napr

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	polygonRingDepth      = 2
	multiPolygonRingDepth = 3
	pointValues           = 2
	// The centroid of a ring is the sum over its edges divided by 6 times its area, and
	// its area is half the shoelace sum.
	centroidFactor = 6
	shoelaceHalf   = 2
)

var errInvalidWKT = errors.New("shape: invalid WKT")

// vertex is a point of a shape: x is the longitude, y the latitude.
type vertex struct{ x, y float64 }

// centre returns the area centroid of the outer rings of a POLYGON or a MULTIPOLYGON,
// weighted by area. A shape with no area gives the mean of its vertices.
func centre(wkt string) (vertex, error) {
	rings, err := outerRings(wkt)
	if err != nil {
		return vertex{}, err
	}

	var (
		area     float64
		sum      vertex
		vertices []vertex
	)

	for _, text := range rings {
		ring, err := parseRing(text)
		if err != nil {
			return vertex{}, err
		}

		ringArea, ringCentre := ringCentroid(ring)
		area += ringArea
		sum.x += ringArea * ringCentre.x
		sum.y += ringArea * ringCentre.y

		vertices = append(vertices, open(ring)...)
	}

	result := mean(vertices)
	if area > 0 {
		result = vertex{x: sum.x / area, y: sum.y / area}
	}

	if !finite(result.x) || !finite(result.y) {
		return vertex{}, fmt.Errorf("%w: no finite centre", errInvalidWKT)
	}

	return result, nil
}

// outerRings returns the text of the first ring of each polygon. A later ring of a
// polygon is a hole.
func outerRings(wkt string) ([]string, error) {
	kind, body, found := strings.Cut(strings.TrimSpace(wkt), "(")
	if !found {
		return nil, fmt.Errorf("%w: no ring", errInvalidWKT)
	}

	scan := scanner{body: "(" + body}

	switch strings.TrimSpace(kind) {
	case "POLYGON":
		scan.ringDepth = polygonRingDepth
	case "MULTIPOLYGON":
		scan.ringDepth = multiPolygonRingDepth
	default:
		return nil, fmt.Errorf("%w: %q is not a polygon", errInvalidWKT, kind)
	}

	return scan.rings()
}

// scanner walks the nested parentheses. A ring opens at ringDepth, and a polygon one
// level above it.
type scanner struct {
	body                    string
	ringDepth               int
	depth, start, ringIndex int
	found                   []string
}

func (s *scanner) rings() ([]string, error) {
	for i, char := range s.body {
		var err error

		switch char {
		case '(':
			err = s.open(i)
		case ')':
			err = s.close(i)
		}

		if err != nil {
			return nil, err
		}
	}

	if s.depth != 0 || len(s.found) == 0 {
		return nil, fmt.Errorf("%w: unbalanced", errInvalidWKT)
	}

	return s.found, nil
}

func (s *scanner) open(at int) error {
	s.depth++

	switch {
	case s.depth > s.ringDepth:
		return fmt.Errorf("%w: too deep", errInvalidWKT)
	case s.depth == s.ringDepth-1:
		s.ringIndex = 0
	case s.depth == s.ringDepth:
		s.start = at + 1
	}

	return nil
}

func (s *scanner) close(at int) error {
	if s.depth == s.ringDepth {
		if s.ringIndex == 0 {
			s.found = append(s.found, s.body[s.start:at])
		}

		s.ringIndex++
	}

	s.depth--

	if s.depth < 0 || (s.depth == 0 && strings.TrimSpace(s.body[at+1:]) != "") {
		return fmt.Errorf("%w: unbalanced", errInvalidWKT)
	}

	return nil
}

func parseRing(text string) ([]vertex, error) {
	parts := strings.Split(text, ",")
	ring := make([]vertex, 0, len(parts))

	for _, part := range parts {
		fields := strings.Fields(part)
		if len(fields) != pointValues {
			return nil, fmt.Errorf("%w: point %q", errInvalidWKT, part)
		}

		lon, lonErr := strconv.ParseFloat(fields[0], 64)
		lat, latErr := strconv.ParseFloat(fields[1], 64)

		if lonErr != nil || latErr != nil || !finite(lon) || !finite(lat) {
			return nil, fmt.Errorf("%w: point %q", errInvalidWKT, part)
		}

		ring = append(ring, vertex{x: lon, y: lat})
	}

	return ring, nil
}

// ringCentroid returns the absolute area and the centroid of a ring. It moves the ring
// to its first vertex first, since the shoelace sum of a small parcel at 44 degrees
// loses its digits.
func ringCentroid(ring []vertex) (float64, vertex) {
	origin := ring[0]

	var cross, sumX, sumY float64

	for i := range len(ring) - 1 {
		x1, y1 := ring[i].x-origin.x, ring[i].y-origin.y
		x2, y2 := ring[i+1].x-origin.x, ring[i+1].y-origin.y
		step := x1*y2 - x2*y1
		cross += step
		sumX += (x1 + x2) * step
		sumY += (y1 + y2) * step
	}

	if cross == 0 {
		return 0, vertex{}
	}

	signedArea := cross / shoelaceHalf

	return math.Abs(signedArea), vertex{
		x: origin.x + sumX/(centroidFactor*signedArea),
		y: origin.y + sumY/(centroidFactor*signedArea),
	}
}

// open drops the last vertex of a ring when it closes the ring.
func open(ring []vertex) []vertex {
	if len(ring) > 1 && ring[0] == ring[len(ring)-1] {
		return ring[:len(ring)-1]
	}

	return ring
}

func mean(vertices []vertex) vertex {
	var sum vertex

	for _, point := range vertices {
		sum.x += point.x
		sum.y += point.y
	}

	count := float64(len(vertices))

	return vertex{x: sum.x / count, y: sum.y / count}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
