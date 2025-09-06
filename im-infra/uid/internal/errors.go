package internal

import "errors"

var (
	ErrInvalidWorkerID     = errors.New("invalid worker ID")
	ErrInvalidDatacenterID = errors.New("invalid datacenter ID")
	ErrClockBackwards      = errors.New("clock moved backwards")
)
