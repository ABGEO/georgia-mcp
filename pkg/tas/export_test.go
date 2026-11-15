package tas

// ToJSON exposes the scanner to the tests.
func ToJSON(js string) (string, error) {
	return toJSON(js)
}
