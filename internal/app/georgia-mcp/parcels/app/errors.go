package app

import "errors"

// ErrParcelNotFound tells that the source has no parcel with the code.
var ErrParcelNotFound = errors.New("parcel: not found")
