package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

// InterfaceTestCase defines a single test case for the HTTP client interface methods.
type InterfaceTestCase struct {
	name        string
	method      string
	url         string
	contentType string
	body        io.Reader
	expectErr   error
}

// TestHTTPClient tests the HTTP client interface methods (Get, Post, Put, Delete, Do)
// using table-driven test cases for happy-path and error conditions.
func TestHTTPClient(t *testing.T) {
	// canned response generator
	createResponse := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{Status: "OK", Code: 200},
			Headers: map[string]*proto.Header{
				"Content-Type": {Values: []string{"application/json"}},
			},
			Body: []byte(`{"message":"success"}`),
		}
		b, _ := pb.Marshal(resp)
		return b
	}

	// Create a mock that returns the canned response and does basic validations
	mock, err := hostmock.New(hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
		Response:           createResponse,
	})
	if err != nil {
		t.Fatalf("failed to create hostmock: %v", err)
	}

	// Create the HTTP client with the mock
	client, err := New(Config{Namespace: "tarmac", HostCall: mock.HostCall})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Define test cases both happy-path and error cases
	tt := []InterfaceTestCase{
		{"GET success", "GET", "http://example.com", "", nil, nil},
		{"GET with bad URL", "GET", "://bad-url", "", nil, ErrInvalidURL},
		{"GET with empty URL", "GET", "", "", nil, ErrInvalidURL},
		{"POST success", "POST", "http://example.com", "application/json", strings.NewReader(`{"x":"y"}`), nil},
		{"POST no body", "POST", "http://example.com", "text/plain", nil, nil},
		{"POST with bad URL", "POST", "://bad-url", "", nil, ErrInvalidURL},
		{"POST with empty URL", "POST", "", "", nil, ErrInvalidURL},
		{"POST with empty content type", "POST", "http://example.com", "", strings.NewReader("body"), nil},
		{"POST with bad reader", "POST", "http://example.com", "text/plain", iotest.ErrReader(TestErrBadReader), TestErrBadReader},
		{"PUT success", "PUT", "http://example.com", "text/plain", strings.NewReader("body"), nil},
		{"PUT with bad URL", "PUT", "://bad-url", "", nil, ErrInvalidURL},
		{"PUT with empty URL", "PUT", "", "", nil, ErrInvalidURL},
		{"PUT with empty content type", "PUT", "http://example.com", "", strings.NewReader("body"), nil},
		{"PUT with bad reader", "PUT", "http://example.com", "text/plain", iotest.ErrReader(TestErrBadReader), TestErrBadReader},
		{"DELETE success", "DELETE", "http://example.com", "", nil, nil},
		{"DELETE with bad URL", "DELETE", "://bad-url", "", nil, ErrInvalidURL},
		{"DELETE with empty URL", "DELETE", "", "", nil, ErrInvalidURL},
		{"PATCH via Do", "PATCH", "http://example.com", "application/json", strings.NewReader(`{"flag":true}`), nil},
		{"DO with bad URL", "PATCH", "://bad-url", "", nil, ErrInvalidURL},
		{"DO with empty URL", "PATCH", "", "", nil, ErrInvalidURL},
	}

	// Run each test case using the appropriate method
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var (
				resp *Response
				err  error
			)

			switch tc.method {
			case "GET":
				resp, err = client.Get(tc.url)
			case "POST":
				resp, err = client.Post(tc.url, tc.contentType, tc.body)
			case "PUT":
				resp, err = client.Put(tc.url, tc.contentType, tc.body)
			case "DELETE":
				resp, err = client.Delete(tc.url)
			default:
				// For other methods, use the Do method this is happy-path only
				req, err := NewRequest(tc.method, tc.url, tc.body)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				// Add content type if specified
				if tc.contentType != "" {
					req.Header.Set("Content-Type", tc.contentType)
				}

				resp, err = client.Do(req)
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})
	}

	// exampleURL
	exampleURL, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatalf("failed to parse example URL: %v", err)
	}

	// Additional test cases for Do method
	tt2 := []struct {
		name        string
		request     *Request
		expectedErr error
	}{
		{"Do with valid request", &Request{Method: "GET", URL: exampleURL}, nil},
		{"Do with empty URL", &Request{Method: "GET"}, ErrInvalidURL},
	}

	for _, tc := range tt2 {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Do(tc.request)
			if err != nil || !errors.Is(err, tc.expectedErr) {
				if tc.expectedErr == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected error %v, got %v", tc.expectedErr, err)
				}
				return
			}
			if resp == nil {
				t.Fatal("expected a response, got nil")
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})
	}
}

type HostMockTestCase struct {
	name             string
	method           string
	url              string
	contentType      string
	body             io.Reader
	mockNamespace    string
	mockCapability   string
	mockFunction     string
	mockResponse     func() []byte
	customHeaders    map[string]string
	expectStatus     string
	expectCode       int
	expectBody       string
	expectErr        bool
	expectErrString  string
	payloadValidator func(payload []byte) error
}

// TestHTTPClientHostMock exercises Get/Post/Put/Delete/Do using hostmock to
// validate protobuf payloads and simulate various host-call outcomes.
func TestHTTPClientHostMock(t *testing.T) {
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
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
		Fail:               true,
		Error:              fmt.Errorf("host call failed"),
	})

	tt := []HostMockTestCase{
		{
			name:           "GET success",
			method:         "GET",
			url:            "http://example.com/api",
			mockNamespace:  "tarmac",
			mockCapability: "httpclient",
			mockFunction:   "call",
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
			mockCapability: "httpclient",
			mockFunction:   "call",
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
			mockCapability: "httpclient",
			mockFunction:   "call",
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
			mockCapability: "httpclient",
			mockFunction:   "call",
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
			mockCapability: "httpclient",
			mockFunction:   "call",
			mockResponse:   createResponseFunc,
			customHeaders: map[string]string{
				"X-API-Key": "test-key",
			},
			payloadValidator: func(payload []byte) error {
				var req proto.HTTPClient
				if err := pb.Unmarshal(payload, &req); err != nil {
					return fmt.Errorf("could not unmarshal payload: %w", err)
				}
				h, ok := req.Headers["X-API-Key"]
				if !ok {
					return fmt.Errorf("header X-API-Key not found")
				}
				// Ensure the header has expected values
				if len(h.Values) == 0 || h.Values[0] != "test-key" {
					return fmt.Errorf("header %s: expected %q, got %v", "X-API-Key", "test-key", h.Values)
				}
				return nil
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
			mockCapability: "httpclient",
			mockFunction:   "call",
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
			mockCapability: "httpclient",
			mockFunction:   "call",
			mockResponse:   createResponseFunc,
			customHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "TarmacSDK/1.0",
				"Accept":        "application/json",
			},
			payloadValidator: func(payload []byte) error {
				var req proto.HTTPClient
				if err := pb.Unmarshal(payload, &req); err != nil {
					return fmt.Errorf("could not unmarshal payload: %w", err)
				}
				for k, v := range map[string]string{
					"Authorization": "Bearer token123",
					"User-Agent":    "TarmacSDK/1.0",
					"Accept":        "application/json",
				} {
					h, ok := req.Headers[k]
					if !ok {
						return fmt.Errorf("header %s not found", k)
					}
					// Ensure the header has expected values
					if len(h.Values) == 0 || h.Values[0] != v {
						return fmt.Errorf("header %s: expected %q, got %v", k, v, h.Values)
					}
				}
				return nil
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var expectedBody []byte
			if tc.body != nil {
				data, _ := io.ReadAll(tc.body)
				expectedBody = data
				tc.body = bytes.NewReader(data)
			}
			baselineValidator := func(payload []byte) error {
				var req proto.HTTPClient
				if err := pb.Unmarshal(payload, &req); err != nil {
					return fmt.Errorf("could not unmarshal payload: %w", err)
				}
				if req.Method != tc.method {
					return fmt.Errorf("method mismatch: expected %s, got %s", tc.method, req.Method)
				}
				if req.Url != tc.url {
					return fmt.Errorf("url mismatch: expected %s, got %s", tc.url, req.Url)
				}
				if req.Body != nil && !bytes.Equal(req.Body, expectedBody) {
					return fmt.Errorf("body mismatch: expected %q, got %q", string(expectedBody), string(req.Body))
				}
				return nil
			}
			// Use the appropriate mock
			var mockHostCall func(string, string, string, []byte) ([]byte, error)

			if tc.name == "Host call failure" {
				mockHostCall = failingMock.HostCall
			} else {
				// Configure a standard mock
				mockCfg := hostmock.Config{
					ExpectedNamespace:  tc.mockNamespace,
					ExpectedCapability: tc.mockCapability,
					ExpectedFunction:   tc.mockFunction,
					PayloadValidator: func(payload []byte) error {
						if err := baselineValidator(payload); err != nil {
							return err
						}
						if tc.payloadValidator != nil {
							return tc.payloadValidator(payload)
						}
						return nil
					},
					Response: tc.mockResponse,
				}
				mock, err := hostmock.New(mockCfg)

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
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
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

var TestErrBadReader = fmt.Errorf("bad reader error")
