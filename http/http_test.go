package http

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"

	"github.com/madflojo/testlazy/things/testurl"
)

var ErrTestBadReader = errors.New("bad reader error")

type InterfaceTestCase struct {
	name        string
	method      string
	url         string
	contentType string
	body        io.Reader
	expectedErr error
}

func TestHTTPClient(t *testing.T) {
	// canned response generator
	createResponse := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{Status: "Host OK", Code: 200},
			Code:   200,
			Headers: map[string]*proto.Header{
				"Content-Type": {Values: []string{"application/json"}},
			},
			Body: []byte(`{"message":"success"}`),
		}
		b, _ := resp.MarshalVT()
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
	client, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"}, HostCall: mock.HostCall})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Run shortcut method tests (Get/Post/Put/Delete and happy-path Do creation)
	t.Run("Shortcuts", func(t *testing.T) {
		tt := []InterfaceTestCase{
			{"GET success", "GET", "http://example.com", "", nil, nil},
			{"GET with bad URL", "GET", "://bad-url", "", nil, ErrInvalidURL},
			{"GET with empty URL", "GET", "", "", nil, ErrInvalidURL},
			{"POST success", "POST", "http://example.com", "application/json", strings.NewReader(`{"x":"y"}`), nil},
			{"POST no body", "POST", "http://example.com", "text/plain", nil, nil},
			{"POST with bad URL", "POST", "://bad-url", "", nil, ErrInvalidURL},
			{"POST with empty URL", "POST", "", "", nil, ErrInvalidURL},
			{"POST with empty content type", "POST", "http://example.com", "", strings.NewReader("body"), nil},
			{
				"POST with bad reader",
				"POST",
				"http://example.com",
				"text/plain",
				iotest.ErrReader(ErrTestBadReader),
				ErrTestBadReader,
			},
			{"PUT success", "PUT", "http://example.com", "text/plain", strings.NewReader("body"), nil},
			{"PUT with bad URL", "PUT", "://bad-url", "", nil, ErrInvalidURL},
			{"PUT with empty URL", "PUT", "", "", nil, ErrInvalidURL},
			{"PUT with empty content type", "PUT", "http://example.com", "", strings.NewReader("body"), nil},
			{
				"PUT with bad reader",
				"PUT",
				"http://example.com",
				"text/plain",
				iotest.ErrReader(ErrTestBadReader),
				ErrTestBadReader,
			},
			{"DELETE success", "DELETE", "http://example.com", "", nil, nil},
			{"DELETE with bad URL", "DELETE", "://bad-url", "", nil, ErrInvalidURL},
			{"DELETE with empty URL", "DELETE", "", "", nil, ErrInvalidURL},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				var (
					resp  *Response
					opErr error
				)

				switch tc.method {
				case "GET":
					resp, opErr = client.Get(tc.url)
				case "POST":
					resp, opErr = client.Post(tc.url, tc.contentType, tc.body)
				case "PUT":
					resp, opErr = client.Put(tc.url, tc.contentType, tc.body)
				case "DELETE":
					resp, opErr = client.Delete(tc.url)
				}

				// Check for expected errors
				if !errors.Is(opErr, tc.expectedErr) {
					t.Fatalf("unexpected error: %v", opErr)
				}

				// If we hit the bad reader path, also ensure ErrReadBody is present
				if opErr != nil && errors.Is(opErr, ErrTestBadReader) && !errors.Is(opErr, ErrReadBody) {
					t.Fatalf("expected ErrReadBody in error chain, got %v", opErr)
				}

				// If no error, ensure we got a valid response
				if opErr == nil {
					if resp == nil {
						t.Fatal("expected non-nil response when no error")
					}
					if resp.StatusCode != http.StatusOK {
						t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
					}
				}
			})
		}
	})

	// Additional test cases for NewRequest
	t.Run("NewRequest", func(t *testing.T) {
		tt := []struct {
			name        string
			method      string
			url         string
			body        io.Reader
			expectedErr error
		}{
			{"Valid PATCH request", "PATCH", "http://example.com", strings.NewReader(`{"flag":true}`), nil},
			{"Bad URL", "PATCH", "://bad-url", nil, ErrInvalidURL},
			{"Empty URL", "PATCH", "", nil, ErrInvalidURL},
			{"Empty Method", "", "http://example.com", nil, ErrInvalidMethod},
			{"Invalid Method", "INVALID_METHOD", "http://example.com", nil, ErrInvalidMethod},
			{"Nil Body", "PATCH", "http://example.com", nil, nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				_, gotErr := NewRequest(tc.method, tc.url, tc.body)
				if !errors.Is(gotErr, tc.expectedErr) {
					t.Fatalf("unexpected error: %v", gotErr)
				}
			})
		}
	})

	// Run Indepth Do method tests
	t.Run("Do", func(t *testing.T) {
		tt := []struct {
			name        string
			request     *Request
			expectedErr error
		}{
			{"Do with nil request", nil, ErrNilRequest},
			{"Do with valid request", &Request{Method: "GET", URL: testurl.URLHTTPS()}, nil},
			{"Do with no shceme URL", &Request{Method: "GET", URL: testurl.URLNoScheme()}, nil},
			{"Do with invalid host URL", &Request{Method: "GET", URL: testurl.URLInvalidHost()}, nil},
			{"Do with no host URL", &Request{Method: "GET", URL: testurl.URLNoHost()}, ErrInvalidURL},
			{"Do with empty URL", &Request{Method: "GET"}, ErrInvalidURL},
			{
				"Do POST with body",
				&Request{
					Method: http.MethodPost,
					URL:    testurl.URLHTTPS(),
					Header: make(http.Header),
					Body:   io.NopCloser(strings.NewReader(`{"x":"y"}`)),
				},
				nil,
			},
			{"Do HEAD", &Request{Method: http.MethodHead, URL: testurl.URLHTTPS()}, nil},
			{"Do OPTIONS", &Request{Method: http.MethodOptions, URL: testurl.URLHTTPS()}, nil},
			{
				"Do with bad reader",
				&Request{
					Method: http.MethodPost,
					URL:    testurl.URLHTTPS(),
					Body:   io.NopCloser(iotest.ErrReader(ErrTestBadReader)),
				},
				ErrReadBody,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				resp2, err2 := client.Do(tc.request)
				if err2 != nil || !errors.Is(err2, tc.expectedErr) {
					if tc.expectedErr == nil {
						t.Fatalf("unexpected error: %v", err2)
					}
					if !errors.Is(err2, tc.expectedErr) {
						t.Errorf("expected error %v, got %v", tc.expectedErr, err2)
					}
					return
				}
				if resp2 == nil {
					t.Fatal("expected a response, got nil")
				}

				if resp2.StatusCode != http.StatusOK {
					t.Errorf("expected status code %d, got %d", http.StatusOK, resp2.StatusCode)
				}
			})
		}
	})

	t.Run("Post without content type omits header", func(t *testing.T) {
		var captured proto.HTTPClient

		client, err := New(Config{
			SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"},
			HostCall: func(namespace, capability, function string, payload []byte) ([]byte, error) {
				if namespace != "tarmac" {
					t.Fatalf("unexpected namespace: %s", namespace)
				}
				if capability != "httpclient" || function != "call" {
					t.Fatalf("unexpected routing: %s/%s", capability, function)
				}

				if err := captured.UnmarshalVT(payload); err != nil {
					t.Fatalf("failed to unmarshal payload: %v", err)
				}

				resp := &proto.HTTPClientResponse{
					Status: &sdkproto.Status{Code: 200},
					Code:   200,
				}
				b, marshalErr := resp.MarshalVT()
				if marshalErr != nil {
					return nil, marshalErr
				}

				return b, nil
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		if _, err := client.Post("http://example.com", "", strings.NewReader("body")); err != nil {
			t.Fatalf("unexpected error posting without content type: %v", err)
		}

		if _, ok := captured.GetHeaders()["Content-Type"]; ok {
			t.Fatalf("expected Content-Type header to be omitted, got %v", captured.GetHeaders())
		}
	})
}
