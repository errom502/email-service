package shared

import "errors"

var (
	ErrPermanentSMTP = errors.New("permanent smtp error")
	ErrTemporarySMTP = errors.New("temporary smtp error")
)
