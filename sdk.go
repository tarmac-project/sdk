package sdk

import (
	"fmt"

	wapc "github.com/wapc/wapc-guest-tinygo"
)

// DefaultNamespace is used when no explicit namespace is provided.
const DefaultNamespace = "tarmac"

var (
	// ErrHandlerNil is returned when the provided function handler is nil.
	ErrHandlerNil = fmt.Errorf("function handler cannot be nil")
)

// Config provides configuration options for SDK initialization.
type Config struct {
	// Namespace controls the function namespace to use for host callbacks.
	// If empty, DefaultNamespace is used.
	Namespace string

	// Handler is the function to be registered as the main WebAssembly entry point.
	Handler func([]byte) ([]byte, error)
}

// RuntimeConfig carries configuration that is used during creation of SDK components.
type RuntimeConfig struct {
	// Namespace is the function namespace used to scope host interactions.
	Namespace string
}

// SDK represents the initialized runtime with a registered waPC handler.
type SDK struct {
	// runtime holds the current runtime configuration snapshot.
	runtime RuntimeConfig

	// handler is the function to be registered as the main WebAssembly entry point.
	handler func([]byte) ([]byte, error)
}

// New initializes the SDK and registers the handler with waPC.
func New(config Config) (*SDK, error) {
	// Validate Handler is not empty
	if config.Handler == nil {
		return nil, ErrHandlerNil
	}

	// Create runtime configuration with defaults
	cfg := RuntimeConfig{Namespace: DefaultNamespace}

	// Override defaults with provided configuration
	if config.Namespace != "" {
		cfg.Namespace = config.Namespace
	}

	// Create SDK instance
	sdk := &SDK{
		runtime: cfg,
		handler: config.Handler,
	}

	// Register the provided handler with waPC
	wapc.RegisterFunction("handler", sdk.handler)

	return sdk, nil
}

// Config returns the current runtime configuration snapshot.
func (s *SDK) Config() RuntimeConfig { return s.runtime }
