package http

import (
	"github.com/tarmac-project/sdk/hostmock"
	"net/http"
	"testing"
)

func TestClient(t *testing.T) {
	mock, err := hostmock.New(hostmock.Config{ExpectedNamespace: "tarmac"})
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	client, err := New(Config{
		HostCall: mock.HostCall,
	})

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("GET request", func(t *testing.T) {
		resp, err := client.Get("http://example.com")
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})

	t.Run("POST request", func(t *testing.T) {
		resp, err := client.Post("http://example.com", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})

	t.Run("PUT request", func(t *testing.T) {
		resp, err := client.Put("http://example.com", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make PUT request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})

	t.Run("DELETE request", func(t *testing.T) {
		resp, err := client.Delete("http://example.com")
		if err != nil {
			t.Fatalf("Failed to make DELETE request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Do request with custom method", func(t *testing.T) {
		req, err := NewRequest("PATCH", "http://example.com", nil)
		if err != nil {
			t.Fatalf("Failed to create PATCH request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make PATCH request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})
}
