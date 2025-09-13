package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"

	"github.com/madflojo/testlazy/things/testurl"
)

var ErrTestBadReader = fmt.Errorf("bad reader error")

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
				}

				// Check for expected errors
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("unexpected error: %v", err)
				}

				// If we hit the bad reader path, also ensure ErrReadBody is present
				if err != nil && errors.Is(err, ErrTestBadReader) && !errors.Is(err, ErrReadBody) {
					t.Fatalf("expected ErrReadBody in error chain, got %v", err)
				}

				// If no error, ensure we got a valid response
				if err == nil {
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
				_, err := NewRequest(tc.method, tc.url, tc.body)
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("unexpected error: %v", err)
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
					Body:   io.NopCloser(iotest.ErrReader(TestErrBadReader)),
				},
				ErrReadBody,
			},
		}

		for _, tc := range tt {
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
	})
}
