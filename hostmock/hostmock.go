package hostmock

import (
    "errors"
    "fmt"
)

var (
    // ErrUnexpectedNamespace is returned when the namespace is not as expected.
    ErrUnexpectedNamespace = errors.New("unexpected namespace")

    // ErrUnexpectedCapability is returned when the capability is not as expected.
    ErrUnexpectedCapability = errors.New("unexpected capability")

    // ErrUnexpectedFunction is returned when the function is not as expected.
    ErrUnexpectedFunction = errors.New("unexpected function")

    // ErrOperationFailed is returned when Fail is set without a custom error.
    ErrOperationFailed = errors.New("operation failed")
)

// Mock simulates a host call interface with validation and configurable responses.
type Mock struct {
    // ExpectedNamespace defines the namespace expected in the host call.
    ExpectedNamespace string

    // ExpectedCapability defines the capability expected in the host call.
    ExpectedCapability string

    // ExpectedFunction defines the function name expected in the host call.
    ExpectedFunction string

    // Error is the error to return if the mock is configured to fail.
    Error error

    // PayloadValidator validates the payload passed to the host call.
    PayloadValidator func([]byte) error

    // Response defines the response to return for the host call.
    Response func() []byte

    // Fail indicates whether the mock should return an error.
    Fail bool
}

// Config represents the configuration for creating a Mock instance.
type Config struct {
    // ExpectedNamespace defines the namespace expected in the host call.
    ExpectedNamespace string

    // ExpectedCapability defines the capability expected in the host call.
    ExpectedCapability string

    // ExpectedFunction defines the function name expected in the host call.
    ExpectedFunction string

    // Error is the error to return if the mock is configured to fail.
    Error error

    // PayloadValidator validates the payload passed to the host call.
    PayloadValidator func([]byte) error

    // Response defines the response to return for the host call.
    Response func() []byte

    // Fail indicates whether the mock should return an error.
    Fail bool
}

// New creates a new instance of the Mock based on the provided Config.
func New(config Config) (*Mock, error) {
	return &Mock{
		ExpectedNamespace:  config.ExpectedNamespace,
		ExpectedCapability: config.ExpectedCapability,
		ExpectedFunction:   config.ExpectedFunction,
		Error:              config.Error,
		Fail:               config.Fail,
		PayloadValidator:   config.PayloadValidator,
		Response:           config.Response,
	}, nil
}

// HostCall simulates a host call, validating inputs and returning a response or error.
func (m *Mock) HostCall(namespace, capability, function string, payload []byte) ([]byte, error) {
	// Return user-defined error if Fail is set
	if m.Fail && m.Error != nil {
		return nil, m.Error
	}

    // Return default error if Fail is set but no custom error is provided
    if m.Fail {
        return nil, ErrOperationFailed
    }

	// Validate namespace
	if m.ExpectedNamespace != namespace {
		return nil, fmt.Errorf(
			"%w: expected namespace %s, got %s",
			ErrUnexpectedNamespace,
			m.ExpectedNamespace,
			namespace,
		)
	}

	// Validate capability
	if m.ExpectedCapability != capability {
		return nil, fmt.Errorf(
			"%w: expected capability %s, got %s",
			ErrUnexpectedCapability,
			m.ExpectedCapability,
			capability,
		)
	}

	// Validate function
	if m.ExpectedFunction != function {
		return nil, fmt.Errorf("%w: expected function %s, got %s", ErrUnexpectedFunction, m.ExpectedFunction, function)
	}

	// Validate payload using user-defined validator, if provided
	if m.PayloadValidator != nil {
		if err := m.PayloadValidator(payload); err != nil {
			return nil, err
		}
	}

	// Return user-defined response if provided
	if m.Response != nil {
		return m.Response(), nil
	}

	// Default to no response
	return nil, nil
}
