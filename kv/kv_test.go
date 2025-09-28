package kv

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/kvstore"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name      string
		namespace string
		wantNS    string
	}{
		{
			name:      "custom namespace",
			namespace: "custom",
			wantNS:    "custom",
		},
		{
			name:      "default namespace",
			namespace: "",
			wantNS:    sdk.DefaultNamespace,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			if client == nil {
				t.Fatal("expected non-nil client")
			}
			if client.Config().Namespace != tc.wantNS {
				t.Fatalf("namespace: want %q got %q", tc.wantNS, client.Config().Namespace)
			}
		})
	}
}

func TestKVInterface(t *testing.T) {
	t.Parallel()

	const (
		namespace  = "interface"
		capability = "kvstore"
	)

	buildHost := func(t *testing.T, configs map[string]hostmock.Config) func(string, string, string, []byte) ([]byte, error) {
		t.Helper()
		mocks := make(map[string]*hostmock.Mock)
		for fn, cfg := range configs {
			cfg.ExpectedNamespace = namespace
			cfg.ExpectedCapability = capability
			cfg.ExpectedFunction = fn
			mock, err := hostmock.New(cfg)
			if err != nil {
				t.Fatalf("hostmock for %s: %v", fn, err)
			}
			mocks[fn] = mock
		}
		return func(ns, capabilityName, fn string, payload []byte) ([]byte, error) {
			mock, ok := mocks[fn]
			if !ok {
				return nil, fmt.Errorf("unexpected host function %q", fn)
			}
			return mock.HostCall(ns, capabilityName, fn, payload)
		}
	}

	tt := []struct {
		name           string
		key            string
		value          []byte
		wantKeys       []string
		expectedErrors map[string]error
		hostConfigs    map[string]hostmock.Config
	}{
		{
			name:     "Valid Key/Value",
			key:      "key1",
			value:    []byte("testdata"),
			wantKeys: []string{"key1"},
			expectedErrors: map[string]error{
				"SET":    nil,
				"GET":    nil,
				"DELETE": nil,
				"KEYS":   nil,
			},
			hostConfigs: map[string]hostmock.Config{
				"set": {
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreSet
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key1" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						if string(req.GetData()) != "testdata" {
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
				"get": {
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreGet
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key1" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						return nil
					},
					Response: func() []byte {
						resp := &proto.KVStoreGetResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Data:   []byte("testdata"),
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				"delete": {
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreDelete
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key1" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						return nil
					},
					Response: func() []byte {
						resp := &proto.KVStoreDeleteResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				"keys": {
					Response: func() []byte {
						resp := &proto.KVStoreKeysResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Keys:   []string{"key1"},
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
			},
		},
		{
			name:     "Empty Key",
			key:      "",
			value:    []byte("different_testdata"),
			wantKeys: []string{},
			expectedErrors: map[string]error{
				"SET":    ErrInvalidKey,
				"GET":    ErrInvalidKey,
				"DELETE": ErrInvalidKey,
				"KEYS":   nil,
			},
			hostConfigs: map[string]hostmock.Config{
				"keys": {
					Response: func() []byte {
						resp := &proto.KVStoreKeysResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Keys:   []string{},
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
			},
		},
		{
			name:     "Empty Value",
			key:      "key3",
			value:    nil,
			wantKeys: []string{"key3"},
			expectedErrors: map[string]error{
				"SET":    ErrInvalidValue,
				"GET":    nil,
				"DELETE": nil,
				"KEYS":   nil,
			},
			hostConfigs: map[string]hostmock.Config{
				"get": {
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreGet
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key3" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						return nil
					},
					Response: func() []byte {
						resp := &proto.KVStoreGetResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}, Data: nil}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				"delete": {
					PayloadValidator: func(payload []byte) error {
						var req proto.KVStoreDelete
						if err := pb.Unmarshal(payload, &req); err != nil {
							return err
						}
						if req.GetKey() != "key3" {
							return fmt.Errorf("unexpected key: %s", req.GetKey())
						}
						return nil
					},
					Response: func() []byte {
						resp := &proto.KVStoreDeleteResponse{Status: &sdkproto.Status{Status: "OK", Code: 0}}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
				"keys": {
					Response: func() []byte {
						resp := &proto.KVStoreKeysResponse{
							Status: &sdkproto.Status{Status: "OK", Code: 0},
							Keys:   []string{"key3"},
						}
						b, _ := pb.Marshal(resp)
						return b
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hostCall := buildHost(t, tc.hostConfigs)
			client, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: namespace}, HostCall: hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			t.Run("SET", func(t *testing.T) {
				setErr := client.Set(tc.key, tc.value)
				if !errors.Is(setErr, tc.expectedErrors["SET"]) {
					t.Fatalf("expected SET error %v, got %v", tc.expectedErrors["SET"], setErr)
				}
			})

			t.Run("GET", func(t *testing.T) {
				_, getErr := client.Get(tc.key)
				if !errors.Is(getErr, tc.expectedErrors["GET"]) {
					t.Fatalf("expected GET error %v, got %v", tc.expectedErrors["GET"], getErr)
				}
			})

			t.Run("DELETE", func(t *testing.T) {
				deleteErr := client.Delete(tc.key)
				if !errors.Is(deleteErr, tc.expectedErrors["DELETE"]) {
					t.Fatalf("expected DELETE error %v, got %v", tc.expectedErrors["DELETE"], deleteErr)
				}
			})

			t.Run("KEYS", func(t *testing.T) {
				keys, keysErr := client.Keys()
				if !errors.Is(keysErr, tc.expectedErrors["KEYS"]) {
					t.Fatalf("expected KEYS error %v, got %v", tc.expectedErrors["KEYS"], keysErr)
				}
				if keysErr == nil && !slices.Equal(keys, tc.wantKeys) {
					t.Fatalf("unexpected keys: got %v, want %v", keys, tc.wantKeys)
				}
			})
		})
	}
}

// TestKVClientHostMock exercises Get, Set, Delete, and Keys using a hostmock to simulate waPC host calls.
func TestKVClientHostMock(t *testing.T) {
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
					Error:              errors.New("host failure"),
				},
				wantValue: nil,
				wantErr:   ErrHostCall,
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
				client, err := New(Config{
					SDKConfig: sdk.RuntimeConfig{Namespace: namespace},
					HostCall:  mock.HostCall,
				})
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
					Error:              errors.New("host failure"),
				},
				wantErr: ErrHostCall,
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
				wantErr: ErrHostResponseInvalid,
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
				client, err := New(Config{
					SDKConfig: sdk.RuntimeConfig{Namespace: namespace},
					HostCall:  mock.HostCall,
				})
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
					Error:              errors.New("host failure"),
				},
				wantErr: ErrHostCall,
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
				client, err := New(Config{
					SDKConfig: sdk.RuntimeConfig{Namespace: namespace},
					HostCall:  mock.HostCall,
				})
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
					Error:              errors.New("host failure"),
				},
				wantKeys: nil,
				wantErr:  ErrHostCall,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				mock, err := hostmock.New(tc.mockConfig)
				if err != nil {
					t.Fatalf("failed to create host mock for keys: %v", err)
				}
				client, err := New(Config{
					SDKConfig: sdk.RuntimeConfig{Namespace: namespace},
					HostCall:  mock.HostCall,
				})
				if err != nil {
					t.Fatalf("failed to create KV client: %v", err)
				}
				gotKeys, err := client.Keys()
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("unexpected error: got %v, want %v", err, tc.wantErr)
				}
				if !slices.Equal(gotKeys, tc.wantKeys) {
					t.Errorf("unexpected keys: got %v, want %v", gotKeys, tc.wantKeys)
				}
			})
		}
	})
}
