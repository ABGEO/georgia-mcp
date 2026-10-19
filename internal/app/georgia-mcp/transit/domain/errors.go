package domain

import "errors"

var (
	// ErrInvalidKind tells that a text is not a kind of stop or of route.
	ErrInvalidKind = errors.New("kind: invalid")
	// ErrInvalidPoint tells that a latitude or a longitude is out of range.
	ErrInvalidPoint = errors.New("point: invalid")
	// ErrInvalidClockTime tells that a text is not a time of H:MM or HH:MM.
	ErrInvalidClockTime = errors.New("clock time: invalid")
	// ErrInvalidTiming tells that a text is not leave_now, depart_at, or arrive_by.
	ErrInvalidTiming = errors.New("timing: invalid")
	// ErrInvalidPreference tells that a text is not a preference of a trip plan.
	ErrInvalidPreference = errors.New("preference: invalid")
	// ErrInvalidTripKinds tells that a trip names bus without minibus, or the reverse.
	ErrInvalidTripKinds = errors.New("trip kinds: bus and minibus apart")
)
