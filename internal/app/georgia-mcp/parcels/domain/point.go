package domain

import "fmt"

const (
	georgiaSouth = 41.0
	georgiaNorth = 43.6
	georgiaWest  = 40.0
	georgiaEast  = 46.8
)

// Point is a place in Georgia, in degrees.
type Point struct {
	Lat float64
	Lon float64
}

// NewPoint returns the point of a latitude and a longitude inside the box of Georgia:
// latitude 41.0 to 43.6, longitude 40.0 to 46.8.
func NewPoint(lat, lon float64) (Point, error) {
	// A comparison with NaN is false, so NaN fails here too.
	if !(lat >= georgiaSouth && lat <= georgiaNorth && lon >= georgiaWest && lon <= georgiaEast) {
		return Point{}, fmt.Errorf("%w: %v, %v", ErrPointOutsideGeorgia, lat, lon)
	}

	return Point{Lat: lat, Lon: lon}, nil
}
