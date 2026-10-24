package app

import "errors"

// ErrInvalidQuery tells that a search text has fewer than 3 or more than 100 characters.
var ErrInvalidQuery = errors.New("query: invalid")
