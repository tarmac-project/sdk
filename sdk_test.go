package sdk

import (
	"errors"
	"testing"
)

type testCase struct {
	name      string
	namespace string
	handler   func(b []byte) ([]byte, error)
	wantErr   error
	wantNs    string
}

func TestNew(t *testing.T) {
	testCases := []testCase{
		{
			name:      "Valid Config",
			namespace: "valid",
			handler:   func(b []byte) ([]byte, error) { return b, nil },
			wantErr:   nil,
			wantNs:    "valid",
		},
		{
			name:      "Empty Namespace",
			namespace: "",
			handler:   func(b []byte) ([]byte, error) { return b, nil },
			wantErr:   nil,
			wantNs:    DefaultNamespace,
		},
		{
			name:      "Nil Handler",
			namespace: "invalid",
			handler:   nil,
			wantErr:   ErrHandlerNil,
			wantNs:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdk, err := New(Config{Namespace: tc.namespace, Handler: tc.handler})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if err != nil {
				return
			}

			t.Run("Check Namespace", func(t *testing.T) {
				if sdk.Config().Namespace != tc.wantNs {
					t.Errorf("expected namespace %q, got %q", tc.wantNs, sdk.Config().Namespace)
				}
			})
		})
	}
}

func TestSDK_Behavior(t *testing.T) {
	// Create two SDK instances up front to cover multiple registrations
	// and enable instance isolation checks.
	h1 := func(b []byte) ([]byte, error) { return b, nil }
	h2 := func(b []byte) ([]byte, error) { return nil, errors.New("boom") }

	s1, err := New(Config{Namespace: "one", Handler: h1})
	if err != nil {
		t.Fatalf("first New returned error: %v", err)
	}
	s2, err := New(Config{Namespace: "two", Handler: h2})
	if err != nil {
		t.Fatalf("second New returned error: %v", err)
	}

	t.Run("MultipleCalls_NoPanic", func(t *testing.T) {
		// If we reached here, both New calls above succeeded without panic.
		if s1 == nil || s2 == nil {
			t.Fatalf("expected non-nil SDK instances")
		}
	})

	t.Run("Config_Immutability", func(t *testing.T) {
		got := s1.Config()
		got.Namespace = "mutated"
		if s1.Config().Namespace != "one" {
			t.Fatalf("expected SDK namespace to remain 'one', got %q", s1.Config().Namespace)
		}
	})

	t.Run("InstancesIsolation", func(t *testing.T) {
		if s1.Config().Namespace != "one" || s2.Config().Namespace != "two" {
			t.Fatalf("expected namespaces 'one' and 'two', got %q and %q", s1.Config().Namespace, s2.Config().Namespace)
		}
	})
}
