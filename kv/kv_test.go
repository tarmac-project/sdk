package kv

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/kvstore"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

// InterfaceTestCase defines a test case structure for KV interface operations
// Each test case includes a name, key-value pair, and expected errors for different operations
type InterfaceTestCase struct {
	Name           string           // Descriptive name of the test case
	Key            string           // Key to use in KV operations
	Value          []byte           // Value to store/retrieve
	ExpectedErrors map[string]error // Map of operation names to expected errors
}

// TestKV tests the functionality of the KV interface implementation
func TestKV(t *testing.T) {
	// Initialize a new KV store with default configuration
	kv, err := New(Config{})
	if err != nil {
		t.Fatalf("Failed to create KV: %v", err)
	}
	defer kv.Close() //nolint:errcheck

	// Define test cases covering different scenarios
	tt := []InterfaceTestCase{
		{
			Name:  "Valid Key/Value",
			Key:   "key1",
			Value: []byte("boring"),
			ExpectedErrors: map[string]error{
				"SET":    nil,
				"GET":    nil,
				"DELETE": nil,
				"KEYS":   nil,
			},
		},
		{
			Name:  "Empty Key",
			Key:   "",
			Value: []byte("less_boring"),
			ExpectedErrors: map[string]error{
				"SET":    ErrInvalidKey,
				"GET":    ErrInvalidKey,
				"DELETE": ErrInvalidKey,
				"KEYS":   nil,
			},
		},
		{
			Name:  "Empty Value",
			Key:   "key3",
			Value: nil,
			ExpectedErrors: map[string]error{
				"SET":    ErrInvalidValue,
				"GET":    nil,
				"DELETE": nil,
				"KEYS":   nil,
			},
		},
	}

	// Run tests for each test case
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			// Test SET operation
			t.Run("SET", func(t *testing.T) {
				err := kv.Set(tc.Key, tc.Value)
				if err != tc.ExpectedErrors["SET"] {
					t.Fatalf("Expected error %v, got %v", tc.ExpectedErrors["SET"], err)
				}
			})

			// Test GET operation
			t.Run("GET", func(t *testing.T) {
				value, err := kv.Get(tc.Key)
				if err != tc.ExpectedErrors["GET"] {
					t.Fatalf("Expected error %v, got %v", tc.ExpectedErrors["GET"], err)
				}
				// Verify returned value matches expected value when no error
				if err == nil && !bytes.Equal(value, tc.Value) {
					t.Fatalf("Expected value %v, got %v", tc.Value, value)
				}
			})

			// Test DELETE operation
			t.Run("DELETE", func(t *testing.T) {
				err := kv.Delete(tc.Key)
				if err != tc.ExpectedErrors["DELETE"] {
					t.Fatalf("Expected error %v, got %v", tc.ExpectedErrors["DELETE"], err)
				}
			})
		})
	}

	// Test KEYS operation separately with a fresh KV instance
	t.Run("KEYS", func(t *testing.T) {
		kv, err := New(Config{})
		if err != nil {
			t.Fatalf("Failed to create KV: %v", err)
		}
		defer kv.Close() //nolint:errcheck

		// Verify Keys() returns expected results
		keys, err := kv.Keys()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(keys) != 5 {
			t.Fatalf("Expected 5 keys, got %d", len(keys))
		}
	})
}

// TestKVHostCall exercises Get, Set, Delete, and Keys using a hostmock to simulate waPC host calls.
func TestKVHostCall(t *testing.T) {
	const namespace = "testing"
	const capability = "kvstore"

	t.Run("Get", func(t *testing.T) {
		tests := []struct {
			name       string
			key        string
			mockConfig hostmock.Config
			wantValue  []byte
			wantErr    error
		}{
			{
				name: "success",
				key:  "key1",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "get",
					Response: func() []byte {
						resp := &proto.KVStoreGetResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Data:   []byte("value1"),
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantValue: []byte("value1"),
				wantErr:   nil,
			},
			{
				name: "host error",
				key:  "key1",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "get",
					Fail:               true,
					Error:              fmt.Errorf("host failure"),
				},
				wantValue: nil,
				wantErr:   fmt.Errorf("host failure"),
			},
			{
				name: "key not found",
				key:  "key2",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "get",
					Response: func() []byte {
						resp := &proto.KVStoreGetResponse{
							Status: &sdkproto.Status{Status: "NotFound", Code: 404},
							Data:   nil,
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantValue: nil,
				wantErr:   ErrKeyNotFound,
			},
			{
				name: "invalid response",
				key:  "key3",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "get",
					Response: func() []byte {
						// Simulating an invalid response that cannot be unmarshalled
						return []byte("invalid response")
					},
				},
				wantValue: nil,
				wantErr:   ErrHostResponseInvalid,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				mock, err := hostmock.New(tc.mockConfig)
				if err != nil {
					t.Fatalf("failed to create host mock: %v", err)
				}
				client, err := New(Config{Namespace: namespace, HostCall: mock.HostCall})
				if err != nil {
					t.Fatalf("failed to create KV client: %v", err)
				}
				gotValue, err := client.Get(tc.key)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("unexpected error: got %v, want %v", err, tc.wantErr)
				}
				if !bytes.Equal(gotValue, tc.wantValue) {
					t.Fatalf("unexpected value: got %v, want %v", gotValue, tc.wantValue)
				}
			})
		}
	})

	t.Run("Set", func(t *testing.T) {
		tests := []struct {
			name       string
			key        string
			value      []byte
			mockConfig hostmock.Config
			wantErr    error
		}{
			{
				name:  "success",
				key:   "key1",
				value: []byte("value1"),
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "set",
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreSet
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key1" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						if string(req.GetData()) != "value1" {
							return fmt.Errorf("unexpected data: %s", string(req.GetData()))
						}
						return nil
					},
					Response: func() []byte {
						resp := &proto.KVStoreSetResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantErr: nil,
			},
			{
				name:  "host error",
				key:   "key1",
				value: []byte("value1"),
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "set",
					Fail:               true,
					Error:              fmt.Errorf("host failure"),
				},
				wantErr: fmt.Errorf("host failure"),
			},
			{
				name:  "invalid payload",
				key:   "key1",
				value: []byte("value1"),
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "set",
					Response: func() []byte {
						resp := &proto.KVStoreSetResponse{Status: &sdkproto.Status{Status: "Invalid", Code: 400}}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantErr: fmt.Errorf("host response invalid"),
			},
			{
				name:  "invalid response",
				key:   "key1",
				value: []byte("value1"),
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "set",
					Response: func() []byte {
						// Simulating an invalid response that cannot be unmarshalled
						return []byte("invalid response")
					},
				},
				wantErr: ErrHostResponseInvalid,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				mock, err := hostmock.New(tc.mockConfig)
				if err != nil {
					t.Fatalf("failed to create host mock for set: %v", err)
				}
				client, err := New(Config{Namespace: namespace, HostCall: mock.HostCall})
				if err != nil {
					t.Fatalf("failed to create KV client: %v", err)
				}
				err = client.Set(tc.key, tc.value)
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("unexpected error: got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("Delete", func(t *testing.T) {
		tests := []struct {
			name       string
			key        string
			mockConfig hostmock.Config
			wantErr    error
		}{
			{
				name: "success",
				key:  "key1",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "delete",
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreDelete
						return pb.Unmarshal(payload, &req)
					},
					Response: func() []byte {
						resp := &proto.KVStoreDeleteResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantErr: nil,
			},
			{
				name: "host error",
				key:  "key1",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "delete",
					Fail:               true,
					Error:              fmt.Errorf("host failure"),
				},
				wantErr: fmt.Errorf("host failure"),
			},
			{
				name: "invalid response",
				key:  "key1",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "delete",
					Response: func() []byte {
						// Simulating an invalid response that cannot be unmarshalled
						return []byte("invalid response")
					},
				},
				wantErr: ErrHostResponseInvalid,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				mock, err := hostmock.New(tc.mockConfig)
				if err != nil {
					t.Fatalf("failed to create host mock for delete: %v", err)
				}
				client, err := New(Config{Namespace: namespace, HostCall: mock.HostCall})
				if err != nil {
					t.Fatalf("failed to create KV client: %v", err)
				}
				err = client.Delete(tc.key)
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("unexpected error: got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("Keys", func(t *testing.T) {
		tests := []struct {
			name       string
			mockConfig hostmock.Config
			wantKeys   []string
			wantErr    error
		}{
			{
				name: "success",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "keys",
					Response: func() []byte {
						resp := &proto.KVStoreKeysResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Keys:   []string{"a", "b", "c"},
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				wantKeys: []string{"a", "b", "c"},
				wantErr:  nil,
			},
			{
				name: "host error",
				mockConfig: hostmock.Config{
					ExpectedNamespace:  namespace,
					ExpectedCapability: capability,
					ExpectedFunction:   "keys",
					Fail:               true,
					Error:              fmt.Errorf("host failure"),
				},
				wantKeys: nil,
				wantErr:  fmt.Errorf("host failure"),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				mock, err := hostmock.New(tc.mockConfig)
				if err != nil {
					t.Fatalf("failed to create host mock for keys: %v", err)
				}
				client, err := New(Config{Namespace: namespace, HostCall: mock.HostCall})
				if err != nil {
					t.Fatalf("failed to create KV client: %v", err)
				}
				gotKeys, err := client.Keys()
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("unexpected error: got %v, want %v", err, tc.wantErr)
				}
				if !equalSlice(gotKeys, tc.wantKeys) {
					t.Errorf("unexpected keys: got %v, want %v", gotKeys, tc.wantKeys)
				}
			})
		}
	})
}

// BenchmarkKV provides basic happy-path benchmarks for Get, Set, Delete, and Keys
// using pre-canned hostmock responses.
func BenchmarkKV(b *testing.B) {
	const namespace = "benchmark"
	const capability = "kvstore"

	// Pre-marshal a happy-path GET response
	getResp := func() []byte {
		resp := &proto.KVStoreGetResponse{
			Status: &sdkproto.Status{Status: "OK", Code: 0},
			Data:   []byte("value"),
		}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockGet, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "get",
		Response:           getResp,
	})
	clientGet, _ := New(Config{Namespace: namespace, HostCall: mockGet.HostCall})

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := clientGet.Get("benchmark-key"); err != nil {
				b.Fatalf("Get failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path SET response
	setResp := func() []byte {
		resp := &proto.KVStoreSetResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockSet, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "set",
		Response:           setResp,
	})
	clientSet, _ := New(Config{Namespace: namespace, HostCall: mockSet.HostCall})

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := clientSet.Set("benchmark-key", []byte("value")); err != nil {
				b.Fatalf("Set failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path DELETE response
	delResp := func() []byte {
		resp := &proto.KVStoreDeleteResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockDel, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "delete",
		Response:           delResp,
	})
	clientDel, _ := New(Config{Namespace: namespace, HostCall: mockDel.HostCall})

	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := clientDel.Delete("benchmark-key"); err != nil {
				b.Fatalf("Delete failed: %v", err)
			}
		}
	})

	// Pre-marshal a happy-path KEYS response
	keysResp := func() []byte {
		resp := &proto.KVStoreKeysResponse{
			Status: &sdkproto.Status{Status: "OK", Code: 0},
			Keys:   []string{"a", "b", "c"},
		}
		bz, _ := pb.Marshal(resp)
		return bz
	}
	mockKeys, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  namespace,
		ExpectedCapability: capability,
		ExpectedFunction:   "keys",
		Response:           keysResp,
	})
	clientKeys, _ := New(Config{Namespace: namespace, HostCall: mockKeys.HostCall})

	b.Run("Keys", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := clientKeys.Keys(); err != nil {
				b.Fatalf("Keys failed: %v", err)
			}
		}
	})
}

// equalSlice compares two string slices for equality.
func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
