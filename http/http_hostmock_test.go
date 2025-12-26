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
)

// Common canned responses used by hostmock tests.
// okResponse returns a standard 200 OK response with JSON body.
func okResponse() []byte {
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
		if err := req.UnmarshalVT(payload); err != nil {
			return fmt.Errorf("could not unmarshal payload: %w", err)
		}
		if req.GetMethod() != method {
			return fmt.Errorf("method mismatch: expected %s, got %s", method, req.GetMethod())
		}
		if req.GetUrl() != url {
			return fmt.Errorf("url mismatch: expected %s, got %s", url, req.GetUrl())
		}
		if expectedBody != nil && !bytes.Equal(req.GetBody(), expectedBody) {
			return fmt.Errorf("body mismatch: expected %q, got %q", string(expectedBody), string(req.GetBody()))
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
					if err := req.UnmarshalVT(p); err != nil {
						return err
					}
					for wantK, wantV := range hv {
						matched := false
						for gotK, h := range req.GetHeaders() {
							if strings.EqualFold(gotK, wantK) {
								matched = h != nil && len(h.GetValues()) > 0 && h.GetValues()[0] == wantV
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
		t.Run(tc.name, func(t *testing.T) {
			mockCfg := hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Fail:               true,
				Error:              errors.New("host call failed"),
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
			if err == nil || !errors.Is(err, sdk.ErrHostCall) {
				t.Fatalf("want sdk.ErrHostCall got %v", err)
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
	// Validate HTTP status code/text derives from response.Code.
	tt := []struct {
		name              string
		httpCode          int
		httpURL           string
		hostStatusCode    int32
		hostStatusMessage string
		expectStatusText  string
		expectErr         error
	}{
		{
			name:              "HTTP 404, host success",
			httpCode:          http.StatusNotFound,
			httpURL:           "http://example.com/missing",
			hostStatusCode:    200,
			hostStatusMessage: "host handled",
			expectStatusText:  http.StatusText(http.StatusNotFound),
		},
		{
			name:              "HTTP 200, host success",
			httpCode:          http.StatusOK,
			httpURL:           "http://example.com/ok",
			hostStatusCode:    200,
			hostStatusMessage: "host ok",
			expectStatusText:  http.StatusText(http.StatusOK),
		},
		{
			name:              "HTTP 0 keeps empty status",
			httpCode:          0,
			httpURL:           "http://example.com/weird",
			hostStatusCode:    200,
			hostStatusMessage: "still host ok",
			expectStatusText:  "",
		},
		{
			name:              "Host error",
			httpCode:          500,
			httpURL:           "http://example.com/fail",
			hostStatusCode:    500,
			hostStatusMessage: "host failure",
			expectErr:         sdk.ErrHostError,
		},
		{
			name:              "Host bad request",
			httpCode:          200,
			httpURL:           "http://example.com/bad",
			hostStatusCode:    400,
			hostStatusMessage: "bad input",
			expectErr:         sdk.ErrHostError,
		},
		{
			name:              "Host response missing status",
			httpCode:          200,
			httpURL:           "http://example.com/missing-status",
			hostStatusCode:    -1,
			hostStatusMessage: "",
			expectErr:         sdk.ErrHostResponseInvalid,
		},
		{
			name:              "Host unknown status code",
			httpCode:          200,
			httpURL:           "http://example.com/unknown-status",
			hostStatusCode:    999,
			hostStatusMessage: "mystery",
			expectErr:         sdk.ErrHostResponseInvalid,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			resp := &proto.HTTPClientResponse{Code: int32(tc.httpCode)}
			if tc.hostStatusCode >= 0 {
				resp.Status = &sdkproto.Status{Status: tc.hostStatusMessage, Code: tc.hostStatusCode}
			}
			b, _ := resp.MarshalVT()
			client, err := newClientWith(hostmock.Config{
				ExpectedNamespace:  sdk.DefaultNamespace,
				ExpectedCapability: "httpclient",
				ExpectedFunction:   "call",
				Response:           func() []byte { return b },
				PayloadValidator:   baselineValidator(http.MethodGet, tc.httpURL, nil),
			})
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			r, err := client.Get(tc.httpURL)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected error %v, got %v", tc.expectErr, err)
			}
			if tc.expectErr != nil {
				return
			}
			if r == nil {
				t.Fatalf("expected response, got nil")
			}
			if r.StatusCode != tc.httpCode {
				t.Fatalf("status code: want %d got %d", tc.httpCode, r.StatusCode)
			}
			if r.Status != tc.expectStatusText {
				t.Fatalf("status text: want %q got %q", tc.expectStatusText, r.Status)
			}
		})
	}
}

func TestHTTPClientHostMock_InsecureFlag(t *testing.T) {
	// Validate that InsecureSkipVerify maps to the protobuf 'Insecure' field.
	validateInsecure := func(expected bool) func([]byte) error {
		return func(p []byte) error {
			var req proto.HTTPClient
			if err := req.UnmarshalVT(p); err != nil {
				return err
			}
			if req.GetInsecure() != expected {
				return fmt.Errorf("insecure flag: want %v got %v", expected, req.GetInsecure())
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
	if _, gerr := c.Get("http://example.com"); gerr != nil {
		t.Fatalf("get: %v", gerr)
	}
	_ = client // silence unused variable in case of future expansion
}

func TestHTTPClientHostMock_NoBodyResponses(t *testing.T) {
	// Response with no body
	statusOnly := func() []byte {
		r := &proto.HTTPClientResponse{Status: &sdkproto.Status{Status: "OK", Code: 200}}
		b, _ := r.MarshalVT()
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
				b, _ := r.MarshalVT()
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
