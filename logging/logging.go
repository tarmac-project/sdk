package logging

import (
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

const capabilityName = "logging"

// Client exposes convenience helpers for sending log entries to the host runtime.
type Client interface {
	Info(message string)
	Warn(message string)
	Error(message string)
	Debug(message string)
	Trace(message string)
}

// Config controls how a Client instance interacts with the host runtime.
type Config struct {
	// SDKConfig provides the runtime namespace used for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for logging operations.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

// HostLogger implements Client using the configured host call entrypoint.
type HostLogger struct {
	runtime  sdk.RuntimeConfig
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// Ensure client implements the Client interface at compile time.
var _ Client = (*HostLogger)(nil)

// New creates a Client that emits logs through the configured host capability.
func New(cfg Config) (*HostLogger, error) {
	runtimeCfg := cfg.SDKConfig
	if runtimeCfg.Namespace == "" {
		runtimeCfg.Namespace = sdk.DefaultNamespace
	}

	hostCall := cfg.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &HostLogger{
		runtime:  runtimeCfg,
		hostCall: hostCall,
	}, nil
}

func (c *HostLogger) Info(message string)  { c.log("Info", message) }
func (c *HostLogger) Warn(message string)  { c.log("Warn", message) }
func (c *HostLogger) Error(message string) { c.log("Error", message) }
func (c *HostLogger) Debug(message string) { c.log("Debug", message) }
func (c *HostLogger) Trace(message string) { c.log("Trace", message) }

func (c *HostLogger) log(fn string, message string) {
	_, _ = c.hostCall(c.runtime.Namespace, capabilityName, fn, []byte(message))
}
