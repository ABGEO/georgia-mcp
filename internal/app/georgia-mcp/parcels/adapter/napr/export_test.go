package napr

// Centre returns the centre of a shape as a longitude and a latitude, for a test.
func Centre(wkt string) (float64, float64, error) {
	middle, err := centre(wkt)

	return middle.x, middle.y, err
}
