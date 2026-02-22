package function

import (
	"bytes"
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
		hostCall    HostCall
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

			c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}, HostCall: tc.hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			if c.runtime.Namespace != tc.wantNS {
				t.Fatalf("namespace mismatch: want %q, got %q", tc.wantNS, c.runtime.Namespace)
			}

			if tc.wantHostPtr != 0 {
				if got := reflect.ValueOf(c.hostCall).Pointer(); got != tc.wantHostPtr {
					t.Fatalf("hostcall pointer mismatch: want %v, got %v", tc.wantHostPtr, got)
				}
			}
		})
	}
}

func TestCall(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name       string
		namespace  string
		fn         string
		input      []byte
		hostCfg    *hostmock.Config
		hostCall   HostCall
		wantOutput []byte
		wantErr    error
	}{
		{
			name:      "happy path",
			namespace: "tarmac",
			fn:        "target-func",
			input:     []byte("payload"),
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   "target-func",
				PayloadValidator: func(payload []byte) error {
					if string(payload) != "payload" {
						return errors.New("payload mismatch")
					}
					return nil
				},
				Response: func() []byte {
					return []byte("result")
				},
			},
			wantOutput: []byte("result"),
		},
		{
			name:    "empty function name",
			fn:      "",
			input:   []byte("payload"),
			wantErr: ErrInvalidFunctionName,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:    "whitespace function name",
			fn:      " \n\t ",
			input:   []byte("payload"),
			wantErr: ErrInvalidFunctionName,
			hostCall: func(string, string, string, []byte) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:      "host error",
			namespace: "tarmac",
			fn:        "target-func",
			input:     []byte("payload"),
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   "target-func",
				Fail:               true,
				Error:              errors.New("boom"),
			},
			wantErr: sdk.ErrHostCall,
		},
		{
			name:      "empty input allowed",
			namespace: "tarmac",
			fn:        "target-func",
			input:     nil,
			hostCfg: &hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   "target-func",
				PayloadValidator: func(payload []byte) error {
					if len(payload) != 0 {
						return errors.New("expected empty payload")
					}
					return nil
				},
				Response: func() []byte {
					return []byte{}
				},
			},
			wantOutput: []byte{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hostCall := tc.hostCall
			if tc.hostCfg != nil {
				mock, err := hostmock.New(*tc.hostCfg)
				if err != nil {
					t.Fatalf("failed to create hostmock: %v", err)
				}
				hostCall = mock.HostCall
			}

			c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}, HostCall: hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			got, gotErr := c.Call(tc.fn, tc.input)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("unexpected error: want %v got %v", tc.wantErr, gotErr)
			}

			if tc.wantErr != nil {
				return
			}

			if !bytes.Equal(got, tc.wantOutput) {
				t.Fatalf("output mismatch: want %q got %q", string(tc.wantOutput), string(got))
			}
		})
	}
}
