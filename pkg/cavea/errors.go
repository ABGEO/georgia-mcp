package cavea

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Cavea answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("cavea: unexpected status")

// APIError is an answer of Cavea with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError

// ErrNotSuccess tells that an answer of the API has the status "error".
var ErrNotSuccess = errors.New("cavea: answer is not a success")

// ErrNoData tells that the data stream of a page holds no list.
var ErrNoData = errors.New("cavea: no data in the page")

// ErrMalformedData tells that an answer or a list of a page does not decode.
var ErrMalformedData = errors.New("cavea: malformed data")

// ErrNoTicket tells that the answer of the seat ticket holds no ticket.
var ErrNoTicket = errors.New("cavea: no seat ticket")

// ErrTooManySessions tells that a call of Availability holds more than
// MaxAvailabilityIDs codes.
var ErrTooManySessions = errors.New("cavea: too many sessions")
