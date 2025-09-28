package kv

import (
	"testing"

	sdk "github.com/tarmac-project/sdk"
)

type InterfaceTestCase struct {
	Name           string           // Descriptive name of the test case
	Key            string           // Key to use in KV operations
	Value          []byte           // Value to store/retrieve
	ExpectedErrors map[string]error // Map of operation names to expected errors
}

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
		tc := tc
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

	kv, err := New(Config{SDKConfig: sdk.RuntimeConfig{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

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
				_, err := kv.Get(tc.Key)
				if err != tc.ExpectedErrors["GET"] {
					t.Fatalf("Expected error %v, got %v", tc.ExpectedErrors["GET"], err)
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
		kv, err := New(Config{SDKConfig: sdk.RuntimeConfig{}})
		if err != nil {
			t.Fatalf("New returned error: %v", err)
		}
		defer kv.Close() //nolint:errcheck

		_, err = kv.Keys()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}
