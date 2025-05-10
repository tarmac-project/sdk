/*
Package mock provides a mock implementation of the HTTP client for testing Tarmac functions.

This package implements the http.Client interface, allowing you to easily test functions
that make HTTP requests without making real network calls. You can configure mock responses
for specific URLs and HTTP methods, or set a default response for any request.

# Basic Usage

Create a mock client with a default response:

	// Create a new mock client with a default response
	mockClient := mock.New(mock.Config{
	    DefaultResponse: &mock.Response{
	        StatusCode: 200,
	        Status: "OK",
	        Body: []byte(`{"success":true}`),
	    },
	})

	// Use in tests
	func TestMyFunction(t *testing.T) {
	    // Pass the mock to your function
	    result := MyFunction(mockClient)
	    // Assert on the result
	    // ...
	}

# Configuring Specific Responses

Set up different responses for specific endpoints:

	// Configure different responses for specific endpoints
	mockClient.On("GET", "https://example.com/api").Return(&mock.Response{
	    StatusCode: 200,
	    Status: "OK",
	    Body: []byte(`{"data":"example"}`),
	})

	mockClient.On("POST", "https://example.com/api").Return(&mock.Response{
	    StatusCode: 201,
	    Status: "Created",
	    Body: []byte(`{"id":123}`),
	})

	// Configure an error response
	mockClient.On("GET", "https://example.com/error").ReturnError(fmt.Errorf("connection refused"))

# Inspecting Requests

After running tests, you can inspect which endpoints were called:

	// Test function that makes HTTP requests
	MyFunction(mockClient)

	// Check what calls were made
	for _, call := range mockClient.Calls {
	    fmt.Printf("Method: %s, URL: %s\n", call.Method, call.URL)
	    // You can also inspect call.Body and call.Header
	}
*/
package mock

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	sdk "github.com/tarmac-project/sdk/http"
)

// MockClient implements the http.Client interface for testing
type MockClient struct {
	// responses maps method+url to predefined responses
	responses map[string]*Response

	// DefaultResponse is returned when no matching response is found
	DefaultResponse *Response

	// Calls tracks which endpoints were called
	Calls []Call
}

// Response represents a mock HTTP response
type Response struct {
	StatusCode int
	Status     string
	Body       []byte
	Header     http.Header
	Error      error
}

// Call represents a recorded call to the mock client
type Call struct {
	Method string
	URL    string
	Body   []byte
	Header http.Header
}

// Config provides configuration for the mock client
type Config struct {
	DefaultResponse *Response
}

// New creates a new mock HTTP client
func New(config Config) *MockClient {
	// Set up default response if not provided
	defaultResp := config.DefaultResponse
	if defaultResp == nil {
		defaultResp = &Response{
			StatusCode: 200,
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

// On registers a mock response for a specific method and URL
func (m *MockClient) On(method, url string) *ResponseBuilder {
	key := method + " " + url
	return &ResponseBuilder{
		client: m,
		key:    key,
	}
}

// Get implements the http.Client interface
func (m *MockClient) Get(url string) (*sdk.Response, error) {
	m.Calls = append(m.Calls, Call{
		Method: "GET",
		URL:    url,
	})

	key := "GET " + url
	resp, found := m.responses[key]
	if !found {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create the response with proper header initialization
	response := &sdk.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Copy the headers if present
	if resp.Header != nil {
		for k, values := range resp.Header {
			for _, v := range values {
				response.Header.Add(k, v)
			}
		}
	}

	return response, nil
}

// Post implements the http.Client interface
func (m *MockClient) Post(url, contentType string, body io.Reader) (*sdk.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	m.Calls = append(m.Calls, Call{
		Method: "POST",
		URL:    url,
		Body:   bodyBytes,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	})

	key := "POST " + url
	resp, found := m.responses[key]
	if !found {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create the response with proper header initialization
	response := &sdk.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Copy the headers if present
	if resp.Header != nil {
		for k, values := range resp.Header {
			for _, v := range values {
				response.Header.Add(k, v)
			}
		}
	}

	return response, nil
}

// Put implements the http.Client interface
func (m *MockClient) Put(url, contentType string, body io.Reader) (*sdk.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	m.Calls = append(m.Calls, Call{
		Method: "PUT",
		URL:    url,
		Body:   bodyBytes,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	})

	key := "PUT " + url
	resp, found := m.responses[key]
	if !found {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create the response with proper header initialization
	response := &sdk.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Copy the headers if present
	if resp.Header != nil {
		for k, values := range resp.Header {
			for _, v := range values {
				response.Header.Add(k, v)
			}
		}
	}

	return response, nil
}

// Delete implements the http.Client interface
func (m *MockClient) Delete(url string) (*sdk.Response, error) {
	m.Calls = append(m.Calls, Call{
		Method: "DELETE",
		URL:    url,
	})

	key := "DELETE " + url
	resp, found := m.responses[key]
	if !found {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create the response with proper header initialization
	response := &sdk.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Copy the headers if present
	if resp.Header != nil {
		for k, values := range resp.Header {
			for _, v := range values {
				response.Header.Add(k, v)
			}
		}
	}

	return response, nil
}

// Do implements the http.Client interface
func (m *MockClient) Do(req *sdk.Request) (*sdk.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	m.Calls = append(m.Calls, Call{
		Method: req.Method,
		URL:    req.URL.String(),
		Body:   bodyBytes,
		Header: req.Header,
	})

	key := req.Method + " " + req.URL.String()
	resp, found := m.responses[key]
	if !found {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create the response with proper header initialization
	response := &sdk.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
	}

	// Copy the headers if present
	if resp.Header != nil {
		for k, values := range resp.Header {
			for _, v := range values {
				response.Header.Add(k, v)
			}
		}
	}

	return response, nil
}

// ResponseBuilder helps build mock responses using a fluent API
type ResponseBuilder struct {
	client *MockClient
	key    string
}

// Return sets the response for the configured method and URL
func (r *ResponseBuilder) Return(response *Response) *MockClient {
	// Ensure header is initialized
	if response.Header == nil {
		response.Header = make(http.Header)
	}

	r.client.responses[r.key] = response
	return r.client
}

// ReturnError configures an error response
func (r *ResponseBuilder) ReturnError(err error) *MockClient {
	r.client.responses[r.key] = &Response{
		Error: err,
	}
	return r.client
}
