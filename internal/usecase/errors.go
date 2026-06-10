package usecase

import "errors"

var (
	ErrEmptyEvent          = errors.New("event is empty")
	ErrVerificationExpired = errors.New("verification expired")
)
