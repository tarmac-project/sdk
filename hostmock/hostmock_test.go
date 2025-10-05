package hostmock

import (
	"bytes"
	"errors"
	"testing"
)

type TestCase struct {
	name       string
	cfg        Config
	payload    []byte
	namespace  string
	capability string
	function   string
	want       []byte
	wantErr    error
}

var ErrMockError = errors.New("Mock error")

func TestHostMock(t *testing.T) {
	tt := []TestCase{
		{
			name: "TestHostMock",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(_ []byte) error {
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("test"),
			want:       []byte("test"),
			wantErr:    nil,
		},
		{
			name: "TestHostMockFail",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Error:              ErrMockError,
				Fail:               true,
				PayloadValidator: func(_ []byte) error {
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("test"),
			want:       []byte("test"),
			wantErr:    ErrMockError,
		},
		{
			name: "Default fail error",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Fail:               true, // no custom Error provided
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("whatever"),
			want:       nil,
			wantErr:    ErrOperationFailed,
		},
		{
			name: "Nil response returns nil",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				// no Response and no validator
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("ok"),
			want:       nil,
			wantErr:    nil,
		},
		{
			name: "Invalid Payload Format",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(payload []byte) error {
					if string(payload) != "valid" {
						return ErrMockError
					}
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("invalid"),
			want:       nil,
			wantErr:    ErrMockError,
		},
		{
			name: "Empty Payload",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(payload []byte) error {
					if len(payload) == 0 {
						return ErrMockError
					}
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte(""),
			want:       nil,
			wantErr:    ErrMockError,
		},
		{
			name: "Unexpected Namespace",
			cfg: Config{
				ExpectedNamespace:  "expected",
				ExpectedCapability: "test",
				ExpectedFunction:   "test",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(_ []byte) error {
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("test"),
			wantErr:    ErrUnexpectedNamespace,
		},
		{
			name: "Unexpected Capability",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "expected",
				ExpectedFunction:   "test",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(_ []byte) error {
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("test"),
			want:       nil,
			wantErr:    ErrUnexpectedCapability,
		},
		{
			name: "Unexpected Function",
			cfg: Config{
				ExpectedNamespace:  "test",
				ExpectedCapability: "test",
				ExpectedFunction:   "expected",
				Error:              nil,
				Fail:               false,
				PayloadValidator: func(_ []byte) error {
					return nil
				},
				Response: func() []byte {
					return []byte("test")
				},
			},
			namespace:  "test",
			capability: "test",
			function:   "test",
			payload:    []byte("test"),
			want:       nil,
			wantErr:    ErrUnexpectedFunction,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := New(tc.cfg)
			if err != nil {
				t.Fatalf("New Mock instance creation failed: %v", err)
			}

			got, err := mock.HostCall(tc.namespace, tc.capability, tc.function, tc.payload)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Mock call returned unexpected error: got %v, want %v", err, tc.wantErr)
			}

			if !bytes.Equal(got, tc.want) {
				t.Fatalf("Mock call returned unexpected response: got %v, want %v", got, tc.want)
			}
		})
	}
}
