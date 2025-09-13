package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

// Common canned responses used by hostmock tests.
// okResponse returns a standard 200 OK response with JSON body.
func okResponse() []byte {
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

// Helpers
// newClientWith builds an HTTP client from a hostmock.Config.
func newClientWith(host hostmock.Config) (Client, error) {
	m, err := hostmock.New(host)
	if err != nil {
		return nil, err
	}
	return New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: host.ExpectedNamespace}, HostCall: m.HostCall})
}

// baselineValidator verifies method, URL, and (optional) body set on the protobuf payload.
func baselineValidator(method, url string, expectedBody []byte) func([]byte) error {
	return func(payload []byte) error {
		var req proto.HTTPClient
		if err := pb.Unmarshal(payload, &req); err != nil {
			return fmt.Errorf("could not unmarshal payload: %w", err)
		}
		if req.Method != method {
			return fmt.Errorf("method mismatch: expected %s, got %s", method, req.Method)
		}
		if req.Url != url {
			return fmt.Errorf("url mismatch: expected %s, got %s", url, req.Url)
		}
		if expectedBody != nil && !bytes.Equal(req.Body, expectedBody) {
			return fmt.Errorf("body mismatch: expected %q, got %q", string(expectedBody), string(req.Body))
		}
		return nil
	}
}

// exec dispatches a request using shortcut methods or Do for custom verbs.
func exec(
	client Client,
	method, url, contentType string,
	body io.Reader,
	headers map[string]string,
) (*Response, error) {
	switch method {
	case http.MethodGet:
		return client.Get(url)
	case http.MethodPost:
		return client.Post(url, contentType, body)
	case http.MethodPut:
		return client.Put(url, contentType, body)
	case http.MethodDelete:
		return client.Delete(url)
	default:
		req, err := NewRequest(method, url, body)
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		return client.Do(req)
	}
}

func TestHTTPClientHostMock_HappyPaths(t *testing.T) {
	// Exercise happy path flows across verbs, including Do with custom headers.
	tt := []struct {
		name         string
		method       string
		url          string
		contentType  string
		body         string
		headers      map[string]string
		expectCode   int
		expectStatus string
		expectBody   string
	}{
		{"GET", http.MethodGet, "http://example.com/api", "", "", nil, 200, "OK", `{"message":"success"}`},
		{
			"POST with body",
			http.MethodPost,
			"http://example.com/api/resource",
			"application/json",
			`{"name":"test"}`,
			nil,
			200,
			"OK",
			`{"message":"success"}`,
		},
		{
			"PUT with body",
			http.MethodPut,
			"http://example.com/api/1",
			"application/json",
			`{"x":1}`,
			nil,
			200,
			"OK",
			`{"message":"success"}`,
		},
		{"DELETE", http.MethodDelete, "http://example.com/api/1", "", "", nil, 200, "OK", `{"message":"success"}`},
		{
			"PATCH with headers",
			http.MethodPatch,
			"http://example.com/api/1",
			"application/json",
			`{"status":"active"}`,
			map[string]string{"X-API-Key": "test-key"},
			200,
			"OK",
			`{"message":"success"}`,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			var bodyBytes []byte
			if tc.body != "" {
				bodyBytes = []byte(tc.body)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			mockCfg := hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				PayloadValidator:   baselineValidator(tc.method, tc.url, bodyBytes),
				Response:           okResponse,
			}
			// Add extra header validation when needed
			if len(tc.headers) > 0 {
				hv := tc.headers
				mockCfg.PayloadValidator = func(p []byte) error {
					if err := baselineValidator(tc.method, tc.url, bodyBytes)(p); err != nil {
						return err
					}
					var req proto.HTTPClient
					if err := pb.Unmarshal(p, &req); err != nil {
						return err
					}
					for wantK, wantV := range hv {
						matched := false
						for gotK, h := range req.Headers {
							if strings.EqualFold(gotK, wantK) {
								matched = h != nil && len(h.Values) > 0 && h.Values[0] == wantV
								break
							}
						}
						if !matched {
							return fmt.Errorf("header %s: expected %q, not found or mismatched", wantK, wantV)
						}
					}
					return nil
				}
			}

			client, err := newClientWith(mockCfg)
			if err != nil {
				t.Fatalf("client: %v", err)
			}

			resp, err := exec(client, tc.method, tc.url, tc.contentType, bodyReader, tc.headers)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tc.expectCode {
				t.Errorf("code: want %d got %d", tc.expectCode, resp.StatusCode)
			}
			if resp.Status != tc.expectStatus {
				t.Errorf("status: want %q got %q", tc.expectStatus, resp.Status)
			}
			if tc.expectBody != "" && resp.Body != nil {
				b, _ := io.ReadAll(resp.Body)
				if string(b) != tc.expectBody {
					t.Errorf("body: want %q got %q", tc.expectBody, string(b))
				}
			}
		})
	}
}

func TestHTTPClientHostMock_HostFailures(t *testing.T) {
	// Simulate hostcall failures across verbs, including Do.
	tt := []struct{ name, method, url, contentType, body string }{
		{"GET fail", http.MethodGet, "http://example.com/a", "", ""},
		{"POST fail", http.MethodPost, "http://example.com/a", "application/json", `{}`},
		{"PUT fail", http.MethodPut, "http://example.com/a", "application/json", `{}`},
		{"DELETE fail", http.MethodDelete, "http://example.com/a", "", ""},
		{"Do PATCH fail", http.MethodPatch, "http://example.com/a", "application/json", `{}`},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockCfg := hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Fail:               true,
				Error:              fmt.Errorf("host call failed"),
			}
			client, err := newClientWith(mockCfg)
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			var rdr io.Reader
			if tc.body != "" {
				rdr = strings.NewReader(tc.body)
			}
			_, err = exec(client, tc.method, tc.url, tc.contentType, rdr, nil)
			if err == nil || !errors.Is(err, ErrHostCall) {
				t.Fatalf("want ErrHostCall got %v", err)
			}
		})
	}
}

func TestHTTPClientHostMock_UnmarshalFailures(t *testing.T) {
	// Ensure invalid protobuf responses are surfaced as ErrUnmarshalResponse.
	tt := []struct{ name, method, url string }{
		{"GET bad protobuf", http.MethodGet, "http://example.com/x"},
		{"POST bad protobuf", http.MethodPost, "http://example.com/x"},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockCfg := hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Response:           func() []byte { return []byte("not-a-protobuf") },
				PayloadValidator:   baselineValidator(tc.method, tc.url, nil),
			}
			client, err := newClientWith(mockCfg)
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			var err2 error
			switch tc.method {
			case http.MethodGet:
				_, err2 = client.Get(tc.url)
			case http.MethodPost:
				_, err2 = client.Post(tc.url, "application/json", strings.NewReader("{}"))
			}
			if err2 == nil || !errors.Is(err2, ErrUnmarshalResponse) {
				t.Fatalf("want ErrUnmarshalResponse got %v", err2)
			}
		})
	}
}

func TestHTTPClientHostMock_StatusCodes(t *testing.T) {
	// Validate that host status/code map correctly to Response fields.
	tt := []struct {
		name        string
		code        int
		status, url string
	}{
		{"200 OK", 200, "OK", "http://example.com/a"},
		{"404 NotFound", 404, "Not Found", "http://example.com/missing"},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := &proto.HTTPClientResponse{Status: &sdkproto.Status{Status: tc.status, Code: int32(tc.code)}}
			b, _ := pb.Marshal(resp)
			client, err := newClientWith(hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Response:           func() []byte { return b },
				PayloadValidator:   baselineValidator(http.MethodGet, tc.url, nil),
			})
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			r, err := client.Get(tc.url)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if r.StatusCode != tc.code || r.Status != tc.status {
				t.Fatalf("want %d/%q got %d/%q", tc.code, tc.status, r.StatusCode, r.Status)
			}
		})
	}
}

func TestHTTPClientHostMock_InsecureFlag(t *testing.T) {
	// Validate that InsecureSkipVerify maps to the protobuf 'Insecure' field.
	validateInsecure := func(expected bool) func([]byte) error {
		return func(p []byte) error {
			var req proto.HTTPClient
			if err := pb.Unmarshal(p, &req); err != nil {
				return err
			}
			if req.Insecure != expected {
				return fmt.Errorf("insecure flag: want %v got %v", expected, req.Insecure)
			}
			return nil
		}
	}

	mockCfg := hostmock.Config{
		ExpectedNamespace:  sdk.DefaultNamespace,
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
		PayloadValidator:   validateInsecure(true),
		Response:           okResponse,
	}
	client, err := newClientWith(mockCfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	// Override to set InsecureSkipVerify = true
	mock, _ := hostmock.New(mockCfg)
	c, err := New(
		Config{
			SDKConfig:          sdk.RuntimeConfig{Namespace: sdk.DefaultNamespace},
			InsecureSkipVerify: true,
			HostCall:           mock.HostCall,
		},
	)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := c.Get("http://example.com"); err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = client // silence unused variable in case of future expansion
}

func TestHTTPClientHostMock_NoBodyResponses(t *testing.T) {
	// Response with no body
	statusOnly := func() []byte {
		r := &proto.HTTPClientResponse{Status: &sdkproto.Status{Status: "OK", Code: 200}}
		b, _ := pb.Marshal(r)
		return b
	}

	tt := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{"POST no response body", http.MethodPost, "http://example.com", `{"x":1}`},
		{"PUT nil request body and no response body", http.MethodPut, "http://example.com/1", ""},
		{"DELETE no response body", http.MethodDelete, "http://example.com/1", ""},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			client, err := newClientWith(hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Response:           statusOnly,
				PayloadValidator: baselineValidator(tc.method, tc.url, func() []byte {
					if tc.body == "" {
						return nil
					}
					return []byte(tc.body)
				}()),
			})
			if err != nil {
				t.Fatalf("client: %v", err)
			}

			var resp *Response
			switch tc.method {
			case http.MethodPost:
				resp, err = client.Post(tc.url, "application/json", strings.NewReader(tc.body))
			case http.MethodPut:
				// nil body path
				resp, err = client.Put(tc.url, "application/json", nil)
			case http.MethodDelete:
				resp, err = client.Delete(tc.url)
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("nil response")
			}
			if resp.Body != nil {
				t.Fatalf("expected no body, got non-nil")
			}
		})
	}

	t.Run("Do HEAD no response body", func(t *testing.T) {
		client, err := newClientWith(hostmock.Config{
			ExpectedNamespace:  sdk.DefaultNamespace,
			ExpectedCapability: "httpclient",
			ExpectedFunction:   "call",
			Response: func() []byte {
				r := &proto.HTTPClientResponse{Status: &sdkproto.Status{Status: "OK", Code: 200}}
				b, _ := pb.Marshal(r)
				return b
			},
			PayloadValidator: baselineValidator(http.MethodHead, "http://example.com", nil),
		})
		if err != nil {
			t.Fatalf("client: %v", err)
		}
		req, err := NewRequest(http.MethodHead, "http://example.com", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		if resp.Body != nil {
			t.Fatalf("expected no body for HEAD, got non-nil")
		}
	})
}
