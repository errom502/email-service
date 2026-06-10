package smtp

import "errors"

var (
	ErrBuildMessage  = errors.New("failed to build Message")
	ErrInvalidTarget = errors.New("invalid target value")
	ErrDeliveryMail  = errors.New("delivery failed")
)
