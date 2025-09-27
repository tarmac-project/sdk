package mock

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdkhttp "github.com/tarmac-project/sdk/http"
)

func TestMockClient(t *testing.T) {
	t.Run("New with default response", func(t *testing.T) {
		client := New(Config{})

		// Verify default response is set
		if client.DefaultResponse == nil {
			t.Fatal("Expected default response to be set")
		}

		if client.DefaultResponse.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", client.DefaultResponse.StatusCode)
		}

		if client.DefaultResponse.Status != "OK" {
			t.Errorf("Expected status OK, got %s", client.DefaultResponse.Status)
		}
	})

	t.Run("New with custom response", func(t *testing.T) {
		customResp := &Response{
			StatusCode: http.StatusCreated,
			Status:     "Created",
			Body:       []byte(`{"id":123}`),
		}

		client := New(Config{
			DefaultResponse: customResp,
		})

		if client.DefaultResponse.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code 201, got %d", client.DefaultResponse.StatusCode)
		}

		if client.DefaultResponse.Status != "Created" {
			t.Errorf("Expected status Created, got %s", client.DefaultResponse.Status)
		}
	})

	t.Run("On and Return", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On(http.MethodGet, "https://example.com/api").Return(&Response{
			StatusCode: 200,
			Status:     "OK",
			Body:       []byte(`{"data":"test"}`),
		})

		// Check if response is stored
		key := http.MethodGet + " https://example.com/api"
		resp, found := client.responses[key]
		if !found {
			t.Fatal("Response not stored for key:", key)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		if string(resp.Body) != `{"data":"test"}` {
			t.Errorf("Expected body %s, got %s", `{"data":"test"}`, string(resp.Body))
		}
	})

	t.Run("ReturnError", func(t *testing.T) {
		client := New(Config{})
		expectedErr := errors.New("connection refused")

		// Configure error response
		client.On(http.MethodGet, "https://example.com/error").ReturnError(expectedErr)

		// Check if error is stored
		key := http.MethodGet + " https://example.com/error"
		resp, found := client.responses[key]
		if !found {
			t.Fatal("Response not stored for key:", key)
		}

		if !errors.Is(resp.Error, expectedErr) {
			t.Errorf("Expected error %v, got %v", expectedErr, resp.Error)
		}
	})

	t.Run("GET with default response", func(t *testing.T) {
		client := New(Config{
			DefaultResponse: &Response{
				StatusCode: 200,
				Status:     "OK",
				Body:       []byte(`{"success":true}`),
			},
		})

		resp, err := client.Get("https://example.com")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		// Verify call was recorded
		if len(client.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(client.Calls))
		}

		if client.Calls[0].Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", client.Calls[0].Method)
		}

		if client.Calls[0].URL != "https://example.com" {
			t.Errorf("Expected URL https://example.com, got %s", client.Calls[0].URL)
		}
	})

	t.Run("GET with custom response", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On("GET", "https://example.com/api").Return(&Response{
			StatusCode: 200,
			Status:     "OK",
			Body:       []byte(`{"data":"test"}`),
		})

		resp, err := client.Get("https://example.com/api")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		if string(body) != `{"data":"test"}` {
			t.Errorf("Expected body %s, got %s", `{"data":"test"}`, string(body))
		}
	})

	t.Run("GET with error", func(t *testing.T) {
		client := New(Config{})
		expectedErr := errors.New("connection refused")

		// Configure error response
		client.On("GET", "https://example.com/error").ReturnError(expectedErr)

		_, err := client.Get("https://example.com/error")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("POST with custom response", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On(http.MethodPost, "https://example.com/api").Return(&Response{
			StatusCode: http.StatusCreated,
			Status:     "Created",
			Body:       []byte(`{"id":123}`),
		})

		resp, err := client.Post("https://example.com/api", "application/json", strings.NewReader(`{"name":"test"}`))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code 201, got %d", resp.StatusCode)
		}

		if string(body) != `{"id":123}` {
			t.Errorf("Expected body %s, got %s", `{"id":123}`, string(body))
		}

		// Verify call was recorded with body
		if len(client.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(client.Calls))
		}

		if string(client.Calls[0].Body) != `{"name":"test"}` {
			t.Errorf("Expected request body %s, got %s", `{"name":"test"}`, string(client.Calls[0].Body))
		}

		// Verify content type was recorded
		if client.Calls[0].Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", client.Calls[0].Header.Get("Content-Type"))
		}
	})

	t.Run("PUT with custom response", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On("PUT", "https://example.com/api/123").Return(&Response{
			StatusCode: 200,
			Status:     "OK",
			Body:       []byte(`{"updated":true}`),
		})

		resp, err := client.Put(
			"https://example.com/api/123",
			"application/json",
			strings.NewReader(`{"name":"updated"}`),
		)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		// Verify call was recorded with body
		if len(client.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(client.Calls))
		}

		if string(client.Calls[0].Body) != `{"name":"updated"}` {
			t.Errorf("Expected request body %s, got %s", `{"name":"updated"}`, string(client.Calls[0].Body))
		}
	})

	t.Run("DELETE with custom response", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On("DELETE", "https://example.com/api/123").Return(&Response{
			StatusCode: 204,
			Status:     "No Content",
		})

		resp, err := client.Delete("https://example.com/api/123")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status code 204, got %d", resp.StatusCode)
		}

		// Verify call was recorded
		if len(client.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(client.Calls))
		}

		if client.Calls[0].Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", client.Calls[0].Method)
		}
	})

	t.Run("Do with custom method", func(t *testing.T) {
		client := New(Config{})

		// Configure endpoint response
		client.On("PATCH", "https://example.com/api/123").Return(&Response{
			StatusCode: 200,
			Status:     "OK",
			Body:       []byte(`{"patched":true}`),
		})

		// Create request
		req, err := sdkhttp.NewRequest("PATCH", "https://example.com/api/123", strings.NewReader(`{"status":"active"}`))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Add custom headers
		req.Header.Set("Authorization", "Bearer token123")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify response
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}

		// Verify call was recorded with body and headers
		if len(client.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(client.Calls))
		}

		if client.Calls[0].Method != http.MethodPatch {
			t.Errorf("Expected method PATCH, got %s", client.Calls[0].Method)
		}

		if string(client.Calls[0].Body) != `{"status":"active"}` {
			t.Errorf("Expected request body %s, got %s", `{"status":"active"}`, string(client.Calls[0].Body))
		}

		if client.Calls[0].Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("Expected Authorization header, got %s", client.Calls[0].Header.Get("Authorization"))
		}
	})

	t.Run("Do with response error handling", func(t *testing.T) {
		client := New(Config{})

		// Configure error response
		client.On(http.MethodGet, "https://example.com/error").ReturnError(errors.New("timeout"))

		// Create request
		req, err := sdkhttp.NewRequest("GET", "https://example.com/error", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		_, err = client.Do(req)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Error() != "timeout" {
			t.Errorf("Expected error timeout, got %v", err)
		}
	})

	t.Run("Client records multiple calls", func(t *testing.T) {
		client := New(Config{})

		// Make multiple requests
		client.Get("https://example.com/1")
		client.Get("https://example.com/2")
		client.Post("https://example.com/3", "application/json", strings.NewReader(`{}`))

		// Verify all calls were recorded
		if len(client.Calls) != 3 {
			t.Fatalf("Expected 3 calls, got %d", len(client.Calls))
		}

		if client.Calls[0].URL != "https://example.com/1" {
			t.Errorf("Expected URL https://example.com/1, got %s", client.Calls[0].URL)
		}

		if client.Calls[1].URL != "https://example.com/2" {
			t.Errorf("Expected URL https://example.com/2, got %s", client.Calls[1].URL)
		}

		if client.Calls[2].URL != "https://example.com/3" {
			t.Errorf("Expected URL https://example.com/3, got %s", client.Calls[2].URL)
		}
	})
}

func TestResponseWithHeaders(t *testing.T) {
	client := New(Config{})

	// Configure response with headers
	client.On(http.MethodGet, "https://example.com/headers").Return(&Response{
		StatusCode: 200,
		Status:     "OK",
		Header: http.Header{
			"Content-Type":  []string{"application/json"},
			"Cache-Control": []string{"no-cache"},
			"X-Api-Version": []string{"1.0"},
		},
		Body: []byte(`{"header_test":true}`),
	})

	resp, err := client.Get("https://example.com/headers")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify headers were set correctly in the Response
	// (Note: The client works with the Header field, not the Headers map)
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control no-cache, got %s", cacheControl)
	}

	apiVersion := resp.Header.Get("X-Api-Version")
	if apiVersion != "1.0" {
		t.Errorf("Expected X-API-Version 1.0, got %s", apiVersion)
	}
}

func TestInvalidRequestBody(t *testing.T) {
	client := New(Config{})

	// Create a reader that will fail on Read
	failingReader := &FailingReader{err: errors.New("read error")}

	// Test POST with failing reader
	_, err := client.Post("https://example.com", "application/json", failingReader)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "read error") {
		t.Errorf("Expected error with 'read error', got %v", err)
	}

	// Test PUT with failing reader
	_, err = client.Put("https://example.com", "application/json", failingReader)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Test Do with failing reader
	req, _ := sdkhttp.NewRequest("PATCH", "https://example.com", failingReader)
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// FailingReader is a mock io.Reader that always returns an error.
type FailingReader struct {
	err error
}

func (f *FailingReader) Read(_ []byte) (int, error) {
	return 0, f.err
}
