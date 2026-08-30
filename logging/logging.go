package logging

import (
	"errors"
	"fmt"

	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

const capabilityName = "logger"

// Level identifies a logging operation supported by the Tarmac host.
type Level string

const (
	// LevelInfo records normal operational messages.
	LevelInfo Level = "info"
	// LevelWarn records conditions that may require attention.
	LevelWarn Level = "warn"
	// LevelError records failed operations.
	LevelError Level = "error"
	// LevelDebug records diagnostic messages.
	LevelDebug Level = "debug"
	// LevelTrace records fine-grained diagnostic messages.
	LevelTrace Level = "trace"
)

// ErrInvalidLevel indicates that Log received an unsupported level.
var ErrInvalidLevel = errors.New("logging level is invalid")

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

func (c *HostLogger) Info(message string)  { _ = c.Log(LevelInfo, message) }
func (c *HostLogger) Warn(message string)  { _ = c.Log(LevelWarn, message) }
func (c *HostLogger) Error(message string) { _ = c.Log(LevelError, message) }
func (c *HostLogger) Debug(message string) { _ = c.Log(LevelDebug, message) }
func (c *HostLogger) Trace(message string) { _ = c.Log(LevelTrace, message) }

// Log sends a message to the host and reports validation or hostcall failures.
func (c *HostLogger) Log(level Level, message string) error {
	switch level {
	case LevelInfo, LevelWarn, LevelError, LevelDebug, LevelTrace:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidLevel, level)
	}

	if _, err := c.hostCall(c.runtime.Namespace, capabilityName, string(level), []byte(message)); err != nil {
		return errors.Join(sdk.ErrHostCall, err)
	}

	return nil
}
