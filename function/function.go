package function

import (
	"errors"
	"strings"

	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

const capabilityName = "function"

// HostCall defines the waPC host function signature used by function calls.
type HostCall func(string, string, string, []byte) ([]byte, error)

// Client defines the functions capability interface.
type Client interface {
	// Call invokes a function route by name and returns its raw output bytes.
	Call(name string, input []byte) ([]byte, error)
}

// Config controls how a Client instance interacts with the host runtime.
type Config struct {
	// SDKConfig provides the runtime namespace used for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for function invocations.
	HostCall HostCall
}

// HostFunction is the functions capability client implementation.
type HostFunction struct {
	runtime  sdk.RuntimeConfig
	hostCall HostCall
}

// Ensure HostFunction satisfies the Client interface at compile time.
var _ Client = (*HostFunction)(nil)

var (
	// ErrInvalidFunctionName indicates an empty or whitespace-only function name.
	ErrInvalidFunctionName = errors.New("function name is invalid")
)

// New creates a functions client with namespace defaults and optional host-call override.
func New(config Config) (*HostFunction, error) {
	runtime := config.SDKConfig
	if runtime.Namespace == "" {
		runtime.Namespace = sdk.DefaultNamespace
	}

	hostCall := config.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &HostFunction{runtime: runtime, hostCall: hostCall}, nil
}

// Call invokes a function route by name and returns its raw output bytes.
func (c *HostFunction) Call(name string, input []byte) ([]byte, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidFunctionName
	}

	resp, err := c.hostCall(c.runtime.Namespace, capabilityName, name, input)
	if err != nil {
		return nil, errors.Join(sdk.ErrHostCall, err)
	}

	return resp, nil
}
