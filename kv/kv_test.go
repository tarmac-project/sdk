package kv

import (
	"bytes"
	"testing"
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
