package logging

import (
	"errors"

	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

// ErrNotImplemented signals that a logging operation is not yet wired to the host.
var ErrNotImplemented = errors.New("logging: not implemented")

// Client exposes convenience helpers for sending log entries to the host runtime.
type Client interface {
	Info(message string) error
	Warn(message string) error
	Error(message string) error
	Debug(message string) error
	Trace(message string) error
}

// Config controls how a Client instance interacts with the host runtime.
type Config struct {
	// SDKConfig provides the runtime namespace used for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for logging operations.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

// client implements Client using the configured host call entrypoint.
type client struct {
	runtime  sdk.RuntimeConfig
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// New creates a Client that emits logs through the configured host capability.
func New(cfg Config) (Client, error) {
	runtimeCfg := cfg.SDKConfig
	if runtimeCfg.Namespace == "" {
		runtimeCfg.Namespace = sdk.DefaultNamespace
	}

	hostCall := cfg.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &client{
		runtime:  runtimeCfg,
		hostCall: hostCall,
	}, nil
}

func (c *client) Info(message string) error  { return ErrNotImplemented }
func (c *client) Warn(message string) error  { return ErrNotImplemented }
func (c *client) Error(message string) error { return ErrNotImplemented }
func (c *client) Debug(message string) error { return ErrNotImplemented }
func (c *client) Trace(message string) error { return ErrNotImplemented }
