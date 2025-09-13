package mock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	sdkhttp "github.com/tarmac-project/sdk/http"
)

// MockClient implements sdkhttp.Client with configurable responses and call
// recording for tests. It never performs network I/O.
//
// revive:disable:exported // Name mirrors package for discoverability; stutter is acceptable here.
type MockClient struct {
	// responses maps "METHOD URL" keys to predefined responses.
	responses map[string]*Response

	// DefaultResponse is returned when no method/URL-specific response exists.
	DefaultResponse *Response

	// Calls records each request observed by the mock client.
	Calls []Call
}

// revive:enable:exported

// Response describes a synthetic HTTP response used by the mock.
type Response struct {
	// StatusCode is the HTTP status code to return.
	StatusCode int
	// Status is the HTTP status text to return.
	Status string
	// Body is the raw payload returned to callers.
	Body []byte
	// Header holds headers to include in the response.
	Header http.Header
	// Error, when set, is returned instead of a successful response.
	Error error
}

// Call captures a single client operation issued through the mock.
type Call struct {
	// Method is the HTTP method used.
	Method string
	// URL is the requested URL string.
	URL string
	// Body contains the request body, if provided.
	Body []byte
	// Header holds request headers passed by the caller.
	Header http.Header
}

// Config controls construction of a MockClient.
type Config struct {
	// DefaultResponse is used when no specific response has been configured.
	DefaultResponse *Response
}

// New creates a new mock HTTP client.
func New(config Config) *MockClient {
	// Set up default response if not provided
	defaultResp := config.DefaultResponse
	if defaultResp == nil {
		defaultResp = &Response{
			StatusCode: http.StatusOK,
			Status:     "OK",
			Body:       []byte(`{"status":"success"}`),
			Header:     make(http.Header),
		}
	}

	// Ensure header is initialized
	if defaultResp.Header == nil {
		defaultResp.Header = make(http.Header)
	}

	return &MockClient{
		responses:       make(map[string]*Response),
		DefaultResponse: defaultResp,
		Calls:           []Call{},
	}
}

// responseFor returns the configured response for method+url or the default.
func (m *MockClient) responseFor(method, url string) *Response {
	key := method + " " + url
	if resp, ok := m.responses[key]; ok {
		return resp
	}
	return m.DefaultResponse
}

// toSDKResponse converts a mock Response into an sdkhttp.Response with copied headers.
func toSDKResponse(r *Response) *sdkhttp.Response {
	resp := &sdkhttp.Response{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(r.Body)),
	}
	if r.Header != nil {
		for k, values := range r.Header {
			for _, v := range values {
				resp.Header.Add(k, v)
			}
		}
	}
	return resp
}

// readAll is a small helper to read request bodies consistently.
func readAll(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	return b, nil
}

// On starts configuration of a response for a given method and URL.
// It returns a builder used to define the returned response or error.
func (m *MockClient) On(method, url string) *ResponseBuilder {
	key := method + " " + url
	return &ResponseBuilder{
		client: m,
		key:    key,
	}
}

// Get records and returns the configured response for a GET request.
func (m *MockClient) Get(url string) (*sdkhttp.Response, error) {
	m.Calls = append(m.Calls, Call{
		Method: "GET",
		URL:    url,
	})

	resp := m.responseFor("GET", url)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return toSDKResponse(resp), nil
}

// Post records and returns the configured response for a POST request.
func (m *MockClient) Post(url, contentType string, body io.Reader) (*sdkhttp.Response, error) {
	bodyBytes, err := readAll(body)
	if err != nil {
		return nil, err
	}

	m.Calls = append(m.Calls, Call{
		Method: "POST",
		URL:    url,
		Body:   bodyBytes,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	})

	resp := m.responseFor("POST", url)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return toSDKResponse(resp), nil
}

// Put records and returns the configured response for a PUT request.
func (m *MockClient) Put(url, contentType string, body io.Reader) (*sdkhttp.Response, error) {
	bodyBytes, err := readAll(body)
	if err != nil {
		return nil, err
	}

	m.Calls = append(m.Calls, Call{
		Method: "PUT",
		URL:    url,
		Body:   bodyBytes,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	})

	resp := m.responseFor("PUT", url)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return toSDKResponse(resp), nil
}

// Delete records and returns the configured response for a DELETE request.
func (m *MockClient) Delete(url string) (*sdkhttp.Response, error) {
	m.Calls = append(m.Calls, Call{
		Method: "DELETE",
		URL:    url,
	})

	resp := m.responseFor("DELETE", url)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return toSDKResponse(resp), nil
}

// Do records and returns the configured response for an arbitrary request.
func (m *MockClient) Do(req *sdkhttp.Request) (*sdkhttp.Response, error) {
	bodyBytes, err := readAll(req.Body)
	if err != nil {
		return nil, err
	}

	m.Calls = append(m.Calls, Call{
		Method: req.Method,
		URL:    req.URL.String(),
		Body:   bodyBytes,
		Header: req.Header,
	})

	resp := m.responseFor(req.Method, req.URL.String())
	if resp.Error != nil {
		return nil, resp.Error
	}
	return toSDKResponse(resp), nil
}

// Compile-time check: ensure MockClient implements the sdkhttp.Client interface.
var _ sdkhttp.Client = (*MockClient)(nil)

// ResponseBuilder helps configure a response for a specific method and URL.
type ResponseBuilder struct {
	client *MockClient
	key    string
}

// Return sets the response for the configured method and URL.
func (r *ResponseBuilder) Return(response *Response) *MockClient {
	// Ensure header is initialized
	if response.Header == nil {
		response.Header = make(http.Header)
	}

	r.client.responses[r.key] = response
	return r.client
}

// ReturnError configures an error response for the configured method and URL.
func (r *ResponseBuilder) ReturnError(err error) *MockClient {
	r.client.responses[r.key] = &Response{
		Error: err,
	}
	return r.client
}
