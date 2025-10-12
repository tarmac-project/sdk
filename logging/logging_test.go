package logging

import (
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

			impl, ok := cli.(*client)
			if !ok {
				t.Fatalf("expected *client implementation, got %T", cli)
			}

			if impl.runtime.Namespace != tc.wantNS {
				t.Fatalf("namespace mismatch: want %q, got %q", tc.wantNS, impl.runtime.Namespace)
			}

			if tc.wantHostPtr != 0 {
				if got := reflect.ValueOf(impl.hostCall).Pointer(); got != tc.wantHostPtr {
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
		{"Info", "Info", func(c Client, msg string) { c.Info(msg) }},
		{"Warn", "Warn", func(c Client, msg string) { c.Warn(msg) }},
		{"Error", "Error", func(c Client, msg string) { c.Error(msg) }},
		{"Debug", "Debug", func(c Client, msg string) { c.Debug(msg) }},
		{"Trace", "Trace", func(c Client, msg string) { c.Trace(msg) }},
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
