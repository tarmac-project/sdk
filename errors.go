package sdk

import "errors"

var (
	// ErrHostCall indicates that a waPC host invocation failed.
	ErrHostCall = errors.New("host call failed")

	// ErrHostResponseInvalid signals that the host returned an invalid or unexpected payload.
	ErrHostResponseInvalid = errors.New("host response is invalid or unexpected")

	// ErrHostError means the host completed the call but reported a failure status.
	ErrHostError = errors.New("host returned an error status")
)
