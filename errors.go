package sdk

import (
	"errors"
	"fmt"
)

var (
	// ErrHostCall indicates that a waPC host invocation failed.
	ErrHostCall = errors.New("host call failed")

	// ErrHostResponseInvalid signals that the host returned an invalid or unexpected payload.
	ErrHostResponseInvalid = errors.New("host response is invalid or unexpected")

	// ErrHostError means the host completed the call but reported a failure status.
	ErrHostError = errors.New("host returned an error status")
)

// HostStatusError indicates the host returned an error status and includes any
// underlying host-call or status cause details.
type HostStatusError struct {
	Capability  string
	Operation   string
	Cause       error
	HostCallErr error
}

// Error returns a human-readable host-status error message.
func (e *HostStatusError) Error() string {
	if e == nil {
		return ErrHostError.Error()
	}

	target := "host operation"
	switch {
	case e.Capability != "" && e.Operation != "":
		target = fmt.Sprintf("%s/%s", e.Capability, e.Operation)
	case e.Capability != "":
		target = e.Capability
	case e.Operation != "":
		target = e.Operation
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", target, ErrHostError, e.Cause)
	}

	return fmt.Sprintf("%s: %s", target, ErrHostError)
}

// Unwrap exposes sentinel and underlying causes to errors.Is/As.
func (e *HostStatusError) Unwrap() []error {
	if e == nil {
		return []error{ErrHostError}
	}

	errs := []error{ErrHostError}
	if e.HostCallErr != nil {
		errs = append(errs, ErrHostCall, e.HostCallErr)
	}
	if e.Cause != nil {
		errs = append(errs, e.Cause)
	}
	return errs
}
