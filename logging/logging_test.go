package logging

import (
	"errors"
	"reflect"
	"testing"

	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	customHostCall := func(string, string, string, []byte) ([]byte, error) {
		return nil, nil
	}

	tt := []struct {
		name        string
		namespace   string
		hostCall    func(string, string, string, []byte) ([]byte, error)
		wantNS      string
		wantHostPtr uintptr
	}{
		{
			name:      "custom namespace",
			namespace: "custom",
			wantNS:    "custom",
		},
		{
			name:        "default namespace with override",
			hostCall:    customHostCall,
			wantNS:      sdk.DefaultNamespace,
			wantHostPtr: reflect.ValueOf(customHostCall).Pointer(),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cli, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}, HostCall: tc.hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			if cli.runtime.Namespace != tc.wantNS {
				t.Fatalf("namespace mismatch: want %q, got %q", tc.wantNS, cli.runtime.Namespace)
			}

			if tc.wantHostPtr != 0 {
				if got := reflect.ValueOf(cli.hostCall).Pointer(); got != tc.wantHostPtr {
					t.Fatalf("hostcall pointer mismatch: want %v, got %v", tc.wantHostPtr, got)
				}
			}
		})
	}
}

func TestClientLogMethods(t *testing.T) {
	t.Parallel()

	const namespace = "loggy"
	message := "mission accomplished"

	tt := []struct {
		name   string
		fn     string
		invoke func(Client, string)
	}{
		{"Info", "info", func(c Client, msg string) { c.Info(msg) }},
		{"Warn", "warn", func(c Client, msg string) { c.Warn(msg) }},
		{"Error", "error", func(c Client, msg string) { c.Error(msg) }},
		{"Debug", "debug", func(c Client, msg string) { c.Debug(msg) }},
		{"Trace", "trace", func(c Client, msg string) { c.Trace(msg) }},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var captured string

			cfg := hostmock.Config{
				ExpectedNamespace:  namespace,
				ExpectedCapability: capabilityName,
				ExpectedFunction:   tc.fn,
				PayloadValidator: func(payload []byte) error {
					captured = string(payload)
					return nil
				},
			}
			mock, err := hostmock.New(cfg)
			if err != nil {
				t.Fatalf("hostmock: %v", err)
			}

			cli, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: mock.HostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			tc.invoke(cli, message)
			if captured != message {
				t.Fatalf("expected captured payload %q, got %q", message, captured)
			}
		})
	}
}

func TestHostLoggerLog(t *testing.T) {
	t.Parallel()

	hostErr := errors.New("host unavailable")
	tests := []struct {
		name          string
		level         Level
		hostErr       error
		wantOperation string
		wantErr       error
		wantCalls     int
	}{
		{name: "info", level: LevelInfo, wantOperation: "info", wantCalls: 1},
		{name: "warn", level: LevelWarn, wantOperation: "warn", wantCalls: 1},
		{name: "error", level: LevelError, wantOperation: "error", wantCalls: 1},
		{name: "debug", level: LevelDebug, wantOperation: "debug", wantCalls: 1},
		{name: "trace", level: LevelTrace, wantOperation: "trace", wantCalls: 1},
		{name: "invalid", level: Level("fatal"), wantErr: ErrInvalidLevel},
		{
			name:          "host failure",
			level:         LevelInfo,
			hostErr:       hostErr,
			wantOperation: "info",
			wantErr:       sdk.ErrHostCall,
			wantCalls:     1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			logger, err := New(Config{
				SDKConfig: sdk.RuntimeConfig{Namespace: "custom"},
				HostCall: func(namespace, capability, operation string, payload []byte) ([]byte, error) {
					calls++
					if namespace != "custom" {
						t.Errorf("namespace = %q, want custom", namespace)
					}
					if capability != "logger" {
						t.Errorf("capability = %q, want logger", capability)
					}
					if operation != tc.wantOperation {
						t.Errorf("operation = %q, want %q", operation, tc.wantOperation)
					}
					if string(payload) != "message" {
						t.Errorf("payload = %q, want message", payload)
					}
					return nil, tc.hostErr
				},
			})
			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}

			err = logger.Log(tc.level, "message")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Log() error = %v, want %v", err, tc.wantErr)
			}
			if tc.hostErr != nil && !errors.Is(err, tc.hostErr) {
				t.Fatalf("Log() error = %v, want host cause %v", err, tc.hostErr)
			}
			if calls != tc.wantCalls {
				t.Fatalf("hostcall count = %d, want %d", calls, tc.wantCalls)
			}
		})
	}
}
