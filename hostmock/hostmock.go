package hostmock

import (
	"fmt"
)

type Mock struct {
	ExpectedNamespace  string
	ExpectedCapability string
	ExpectedFunction   string
	Error              error
	Fail               bool

	PayloadValidator func([]byte) error
	Response         func() []byte
}

type Config struct {
	ExpectedNamespace  string
	ExpectedCapability string
	ExpectedFunction   string
	Error              error
	Fail               bool

	PayloadValidator func([]byte) error
	Response         func() []byte
}

var ErrUnexpectedNamespace = fmt.Errorf("Unexpected namespace")
var ErrUnexpectedCapability = fmt.Errorf("Unexpected capability")
var ErrUnexpectedFunction = fmt.Errorf("Unexpected function")

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

func (m *Mock) HostCall(namespace, capability, function string, payload []byte) ([]byte, error) {
	// Return user provided error
	if m.Fail && m.Error != nil {
		return nil, m.Error
	}

	// Return default error if Fail is true but no error is provided
	if m.Fail {
		return nil, fmt.Errorf("Failed")
	}

	// Validate the call parameters
	if m.ExpectedNamespace != namespace {
		return nil, fmt.Errorf("%w: Expected namespace %s, got %s", ErrUnexpectedNamespace, m.ExpectedNamespace, namespace)
	}

	if m.ExpectedCapability != capability {
		return nil, fmt.Errorf("%w: Expected capability %s, got %s", ErrUnexpectedCapability, m.ExpectedCapability, capability)
	}

	if m.ExpectedFunction != function {
		return nil, fmt.Errorf("%w: Expected function %s, got %s", ErrUnexpectedFunction, m.ExpectedFunction, function)
	}

	// Ensure the payload is valid
	if m.PayloadValidator != nil {
		if err := m.PayloadValidator(payload); err != nil {
			return nil, err
		}
	}

	// Return the user provided response if it exists
	if m.Response != nil {
		return m.Response(), nil
	}

	// Return default response of nothing
	return nil, nil
}
