package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

func TestClient(t *testing.T) {
	// Create a mock response generator
	createResponseFunc := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{
				Status: "OK",
				Code:   200,
			},
			Headers: map[string]*proto.Header{
				"Content-Type": {
					Values: []string{"application/json"},
				},
			},
			Body: []byte(`{"message":"success"}`),
		}

		respBytes, _ := pb.Marshal(resp)
		return respBytes
	}

	// Configure the mock with a response
	mock, err := hostmock.New(hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: "http",
		ExpectedFunction:   "http",
		Response:           createResponseFunc,
	})

	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	// Create the client with proper namespace
	client, err := New(Config{
		Namespace: "tarmac",
		HostCall:  mock.HostCall,
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

		if resp.Status != "OK" {
			t.Fatalf("Expected status text 'OK', got %v", resp.Status)
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Expected Content-Type application/json, got %v", contentType)
		}

		// Test body if present
		if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			expectedBody := `{"message":"success"}`
			if string(body) != expectedBody {
				t.Fatalf("Expected body %q, got %q", expectedBody, string(body))
			}
		}
	})

	t.Run("POST request", func(t *testing.T) {
		resp, err := client.Post("http://example.com", "application/json", strings.NewReader(`{"data":"test"}`))
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})

	t.Run("PUT request", func(t *testing.T) {
		resp, err := client.Put("http://example.com", "application/json", strings.NewReader(`{"data":"test"}`))
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

		// Add some headers to test header processing
		req.Header.Set("X-Custom-Header", "test-value")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make PATCH request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK, got %v", resp.StatusCode)
		}
	})
}

func TestClientTable(t *testing.T) {
	// Create a mock response generator
	createResponseFunc := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{
				Status: "OK",
				Code:   200,
			},
			Headers: map[string]*proto.Header{
				"Content-Type": {
					Values: []string{"application/json"},
				},
				"X-Rate-Limit": {
					Values: []string{"100"},
				},
			},
			Body: []byte(`{"message":"success"}`),
		}

		respBytes, _ := pb.Marshal(resp)
		return respBytes
	}

	// Create a mock for error cases
	createErrorResponseFunc := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{
				Status: "Not Found",
				Code:   404,
			},
			Headers: map[string]*proto.Header{
				"Content-Type": {
					Values: []string{"application/json"},
				},
			},
			Body: []byte(`{"error":"resource not found"}`),
		}

		respBytes, _ := pb.Marshal(resp)
		return respBytes
	}

	// Create a mock that fails
	failingMock, _ := hostmock.New(hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: "http",
		ExpectedFunction:   "http",
		Fail:               true,
		Error:              fmt.Errorf("host call failed"),
	})

	tests := []struct {
		name            string
		method          string
		url             string
		contentType     string
		body            io.Reader
		mockNamespace   string
		mockCapability  string
		mockFunction    string
		mockResponse    func() []byte
		customHeaders   map[string]string
		expectStatus    string
		expectCode      int
		expectBody      string
		expectErr       bool
		expectErrString string
	}{
		{
			name:           "GET success",
			method:         "GET",
			url:            "http://example.com/api",
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			expectStatus:   "OK",
			expectCode:     200,
			expectBody:     `{"message":"success"}`,
		},
		{
			name:           "POST with body",
			method:         "POST",
			url:            "http://example.com/api/resource",
			contentType:    "application/json",
			body:           strings.NewReader(`{"name":"test"}`),
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			expectStatus:   "OK",
			expectCode:     200,
			expectBody:     `{"message":"success"}`,
		},
		{
			name:           "PUT with body",
			method:         "PUT",
			url:            "http://example.com/api/resource/123",
			contentType:    "application/json",
			body:           strings.NewReader(`{"name":"updated"}`),
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			expectStatus:   "OK",
			expectCode:     200,
			expectBody:     `{"message":"success"}`,
		},
		{
			name:           "DELETE resource",
			method:         "DELETE",
			url:            "http://example.com/api/resource/123",
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			expectStatus:   "OK",
			expectCode:     200,
			expectBody:     `{"message":"success"}`,
		},
		{
			name:           "Custom PATCH method",
			method:         "PATCH",
			url:            "http://example.com/api/resource/123",
			contentType:    "application/json",
			body:           strings.NewReader(`{"status":"active"}`),
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			customHeaders: map[string]string{
				"X-API-Key":    "test-key",
				"X-Request-ID": "test-request-id",
			},
			expectStatus: "OK",
			expectCode:   200,
			expectBody:   `{"message":"success"}`,
		},
		{
			name:           "404 Not Found Response",
			method:         "GET",
			url:            "http://example.com/api/nonexistent",
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createErrorResponseFunc,
			expectStatus:   "Not Found",
			expectCode:     404,
			expectBody:     `{"error":"resource not found"}`,
		},
		{
			name:            "Custom request with bad URL",
			method:          "CUSTOM",
			url:             "://bad-url",
			mockNamespace:   "tarmac",
			mockCapability:  "http",
			mockFunction:    "http",
			mockResponse:    createResponseFunc,
			expectErr:       true,
			expectErrString: "missing protocol scheme",
		},
		{
			name:           "Custom headers in request",
			method:         "GET",
			url:            "http://example.com/api/headers",
			mockNamespace:  "tarmac",
			mockCapability: "http",
			mockFunction:   "http",
			mockResponse:   createResponseFunc,
			customHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "TarmacSDK/1.0",
				"Accept":        "application/json",
			},
			expectStatus: "OK",
			expectCode:   200,
			expectBody:   `{"message":"success"}`,
		},
		{
			name:            "Host call failure",
			method:          "GET",
			url:             "http://example.com/error",
			mockNamespace:   "tarmac",
			mockCapability:  "http",
			mockFunction:    "http",
			expectErr:       true,
			expectErrString: "host returned error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the appropriate mock
			var mockHostCall func(string, string, string, []byte) ([]byte, error)

			if tc.name == "Host call failure" {
				mockHostCall = failingMock.HostCall
			} else {
				// Configure a standard mock
				mock, err := hostmock.New(hostmock.Config{
					ExpectedNamespace:  tc.mockNamespace,
					ExpectedCapability: tc.mockCapability,
					ExpectedFunction:   tc.mockFunction,
					Response:           tc.mockResponse,
				})

				if err != nil {
					t.Fatalf("Failed to create mock: %v", err)
				}

				mockHostCall = mock.HostCall
			}

			// Create the client
			client, err := New(Config{
				Namespace: tc.mockNamespace,
				HostCall:  mockHostCall,
			})

			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			var resp *Response
			var reqErr error

			// Execute the appropriate method based on the test case
			switch tc.method {
			case "GET":
				resp, reqErr = client.Get(tc.url)
			case "POST":
				resp, reqErr = client.Post(tc.url, tc.contentType, tc.body)
			case "PUT":
				resp, reqErr = client.Put(tc.url, tc.contentType, tc.body)
			case "DELETE":
				resp, reqErr = client.Delete(tc.url)
			default:
				// For other methods, use the Do method
				req, err := NewRequest(tc.method, tc.url, tc.body)
				if err != nil {
					// If we're expecting a URL parse error, this is fine
					if tc.expectErr && tc.expectErrString != "" && strings.Contains(err.Error(), tc.expectErrString) {
						reqErr = err
						return
					}
					t.Fatalf("Failed to create request: %v", err)
				}

				// Add custom headers if specified
				if tc.customHeaders != nil {
					for k, v := range tc.customHeaders {
						req.Header.Set(k, v)
					}
				}

				if tc.contentType != "" {
					req.Header.Set("Content-Type", tc.contentType)
				}

				resp, reqErr = client.Do(req)
			}

			// Check for errors
			if tc.expectErr {
				if reqErr == nil {
					t.Errorf("Expected error but got nil")
				} else if tc.expectErrString != "" && !strings.Contains(reqErr.Error(), tc.expectErrString) {
					t.Errorf("Expected error to contain %q but got %q", tc.expectErrString, reqErr.Error())
				}
				return
			} else if reqErr != nil {
				t.Fatalf("Unexpected error: %v", reqErr)
			}

			// Verify status and code
			if resp.Status != tc.expectStatus {
				t.Errorf("Expected status %q but got %q", tc.expectStatus, resp.Status)
			}

			if resp.StatusCode != tc.expectCode {
				t.Errorf("Expected status code %d but got %d", tc.expectCode, resp.StatusCode)
			}

			// Check Content-Type header
			if tc.contentType != "" {
				if respContentType := resp.Header.Get("Content-Type"); respContentType != "application/json" {
					t.Errorf("Expected Content-Type %q but got %q", "application/json", respContentType)
				}
			}

			// Verify body if expected
			if tc.expectBody != "" && resp.Body != nil {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				if string(body) != tc.expectBody {
					t.Errorf("Expected body %q but got %q", tc.expectBody, string(body))
				}
			}
		})
	}
}

func BenchmarkHTTPClient(b *testing.B) {
	// Create a mock response generator
	createResponseFunc := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{
				Status: "OK",
				Code:   200,
			},
			Headers: map[string]*proto.Header{
				"Content-Type": {
					Values: []string{"application/json"},
				},
			},
			Body: []byte(`{"message":"success"}`),
		}

		respBytes, _ := pb.Marshal(resp)
		return respBytes
	}

	// Configure the mock
	mock, err := hostmock.New(hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: "http",
		ExpectedFunction:   "http",
		Response:           createResponseFunc,
	})

	if err != nil {
		b.Fatalf("Failed to create mock: %v", err)
	}

	// Create the client
	client, err := New(Config{
		Namespace: "tarmac",
		HostCall:  mock.HostCall,
	})

	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	b.Run("GET", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Get("http://example.com")
			if err != nil {
				b.Fatalf("Failed to make GET request: %v", err)
			}
		}
	})

	b.Run("POST", func(b *testing.B) {
		data := strings.NewReader(`{"data":"test"}`)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Reset the reader position for each iteration
			data.Reset(`{"data":"test"}`)
			_, err := client.Post("http://example.com", "application/json", data)
			if err != nil {
				b.Fatalf("Failed to make POST request: %v", err)
			}
		}
	})
}
