package logging

import (
	"errors"
	"reflect"
	"testing"

	sdk "github.com/tarmac-project/sdk"
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}, HostCall: tc.hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			impl, ok := client.(*client)
			if !ok {
				t.Fatalf("expected *client implementation, got %T", client)
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

func TestClientMethodsNotImplemented(t *testing.T) {
	t.Parallel()

	client, err := New(Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	tt := []struct {
		name string
		call func(Client) error
	}{
		{"Info", func(c Client) error { return c.Info("msg") }},
		{"Warn", func(c Client) error { return c.Warn("msg") }},
		{"Error", func(c Client) error { return c.Error("msg") }},
		{"Debug", func(c Client) error { return c.Debug("msg") }},
		{"Trace", func(c Client) error { return c.Trace("msg") }},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.call(client), ErrNotImplemented) {
				t.Fatalf("expected ErrNotImplemented for %s", tc.name)
			}
		})
	}
}
