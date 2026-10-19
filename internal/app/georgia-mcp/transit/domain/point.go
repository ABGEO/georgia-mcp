package domain

import (
	"fmt"
	"math"
)

const (
	maxLatitude  = 90
	maxLongitude = 180
	earthRadius  = 6_371_008.8

	// The box of Tbilisi is the box that the web application of the TTC sends to its
	// place search.
	tbilisiSouth = 41.59149226
	tbilisiNorth = 41.84319686
	tbilisiWest  = 44.66221063
	tbilisiEast  = 45.050854455
)

// Point is a place on the earth, in degrees.
type Point struct {
	Lat float64
	Lon float64
}

// NewPoint returns the point of a latitude and a longitude.
func NewPoint(lat, lon float64) (Point, error) {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.Abs(lat) > maxLatitude ||
		math.Abs(lon) > maxLongitude {
		return Point{}, fmt.Errorf("%w: %v, %v", ErrInvalidPoint, lat, lon)
	}

	return Point{Lat: lat, Lon: lon}, nil
}

// InTbilisi reports whether the point is inside the box that the TTC serves.
func (p Point) InTbilisi() bool {
	return p.Lat >= tbilisiSouth && p.Lat <= tbilisiNorth &&
		p.Lon >= tbilisiWest && p.Lon <= tbilisiEast
}

// MetresTo returns the distance to another point along the surface of the earth.
func (p Point) MetresTo(other Point) float64 {
	lat1, lat2 := radians(p.Lat), radians(other.Lat)
	deltaLat := lat2 - lat1
	deltaLon := radians(other.Lon - p.Lon)

	sinLat := math.Sin(deltaLat / 2) //nolint:mnd // The half angle of the haversine.
	sinLon := math.Sin(deltaLon / 2) //nolint:mnd // The half angle of the haversine.
	haversine := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon

	return 2 * earthRadius * math.Asin(math.Sqrt(haversine))
}

// Box is a box on the map, in degrees.
type Box struct {
	West  float64
	North float64
	East  float64
	South float64
}

// TbilisiBox returns the box that the TTC serves.
func TbilisiBox() Box {
	return Box{West: tbilisiWest, North: tbilisiNorth, East: tbilisiEast, South: tbilisiSouth}
}

func radians(degrees float64) float64 {
	return degrees * math.Pi / 180 //nolint:mnd // The degrees of a half turn.
}
