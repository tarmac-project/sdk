/*
Package http provides a client for making HTTP requests from WebAssembly functions running in Tarmac.

This package allows Tarmac functions to make outbound HTTP requests to external services. It uses the
Web Assembly Procedure Call (waPC) protocol to communicate with the Tarmac host, which handles the
actual HTTP communication.

# Basic Usage

Create a client and make requests:

	client, err := http.New(http.Config{
	    Namespace: "my-service",
	})
	if err != nil {
	    // handle error
	}

	// Make a GET request
	resp, err := client.Get("https://example.com")
	if err != nil {
	    // handle error
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
	    // handle error
	}

	// Use the response data
	fmt.Println(string(data))

# Making POST/PUT Requests

Send data with POST/PUT requests:

	// POST request with JSON body
	jsonBody := strings.NewReader(`{"key":"value"}`)
	resp, err := client.Post("https://example.com/api", "application/json", jsonBody)
	if err != nil {
	    // handle error
	}

	// PUT request
	resp, err := client.Put("https://example.com/resource/123", "application/json", jsonBody)
	if err != nil {
	    // handle error
	}

# Custom Requests

Create custom requests with headers:

	// Create a custom request
	req, err := http.NewRequest("PATCH", "https://example.com/resource", jsonBody)
	if err != nil {
	    // handle error
	}

	// Add custom headers
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom-Header", "value")

	// Send the custom request
	resp, err := client.Do(req)
	if err != nil {
	    // handle error
	}

# Testing with Mocks

The http/mock subpackage provides a simple way to mock HTTP responses for testing:

	import "github.com/tarmac-project/sdk/http/mock"

	// Create a mock HTTP client
	mockClient := mock.New(mock.Config{
	    DefaultResponse: &mock.Response{
	        StatusCode: 200,
	        Status: "OK",
	        Body: []byte(`{"success":true}`),
	    },
	})

	// Configure specific endpoint responses
	mockClient.On("GET", "https://example.com/api").Return(&mock.Response{
	    StatusCode: 200,
	    Status: "OK",
	    Body: []byte(`{"data":"example"}`),
	})

	// Configure an error response
	mockClient.On("GET", "https://example.com/error").ReturnError(fmt.Errorf("connection failed"))
*/
package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
	pb "google.golang.org/protobuf/proto"
)

// Client provides an interface for making HTTP requests
type Client interface {
	Get(url string) (*Response, error)
	Post(url, contentType string, body io.Reader) (*Response, error)
	Put(url, contentType string, body io.Reader) (*Response, error)
	Delete(url string) (*Response, error)
	Do(req *Request) (*Response, error)
}

// Config provides configuration options for the HTTP client
type Config struct {
	// Namespace controls the function namespace to use for host callbacks
	// The default value is "default" which is the global namespace
	Namespace string

	// SDKConfig supplies shared SDK-level configuration such as the
	// default Namespace. If Namespace above is set, it takes precedence.
	SDKConfig sdk.RuntimeConfig

	// InsecureSkipVerify controls whether the client verifies the
	// server's certificate chain and host name
	InsecureSkipVerify bool

	// HostCall is used internally for host callbacks
	// This is mainly here for testing
	HostCall func(string, string, string, []byte) ([]byte, error)
}

type httpClient struct {
	cfg      Config
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// Response represents an HTTP response
type Response struct {
	Status     string
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

// Request represents an HTTP request to be sent by the client
type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   io.ReadCloser
}

var (
	// ErrorInvalidURL is returned when the provided URL is invalid
	ErrInvalidURL = errors.New("invalid URL provided")

	// ErrorReadBody is returned when reading the request body fails
	ErrReadBody = errors.New("failed to read request body")

	// ErrorUnmarshalResponse is returned when unmarshalling the response fails
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")

	// ErrorHostCall is returned when the host call fails
	ErrHostCall = errors.New("host call failed")
)

// New creates a new HTTP client with the provided configuration
func New(config Config) (Client, error) {
	// Set default namespace if not provided
	if config.Namespace == "" {
		if config.SDKConfig.Namespace != "" {
			config.Namespace = config.SDKConfig.Namespace
		} else {
			config.Namespace = "default"
		}
	}

	// Use the provided host call function or default to waPC.HostCall
	hostCallFn := config.HostCall
	if hostCallFn == nil {
		hostCallFn = wapc.HostCall
	}

	return &httpClient{
		hostCall: hostCallFn,
		cfg:      config,
	}, nil
}

func (c *httpClient) Get(url string) (*Response, error) {
	req := &proto.HTTPClient{
		Method:   "GET",
		Url:      url,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}

	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.hostCall(c.cfg.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, fmt.Errorf("host returned error: %w", err)
	}

	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, fmt.Errorf("failed to unmarshal host response: %w", err)
	}

	// Build the response object
	response := &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
		Header:     make(http.Header),
	}

	// Convert headers if present
	for name, header := range r.Headers {
		response.Header[name] = header.Values
	}

	// Add body if present
	if len(r.Body) > 0 {
		response.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return response, nil
}

func (c *httpClient) Post(url, contentType string, body io.Reader) (*Response, error) {
	// Read the body content if present
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return &Response{}, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "POST",
		Url:      url,
		Insecure: c.cfg.InsecureSkipVerify,
		Body:     bodyBytes,
		Headers: map[string]*proto.Header{
			"Content-Type": {
				Values: []string{contentType},
			},
		},
	}

	// Marshal the request
	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, fmt.Errorf("host returned error: %w", err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, fmt.Errorf("failed to unmarshal host response: %w", err)
	}

	// Build the response object
	response := &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
		Header:     make(http.Header),
	}

	// Convert headers if present
	for name, header := range r.Headers {
		response.Header[name] = header.Values
	}

	// Add body if present
	if len(r.Body) > 0 {
		response.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return response, nil
}

func (c *httpClient) Put(url, contentType string, body io.Reader) (*Response, error) {
	// Read the body content if present
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return &Response{}, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "PUT",
		Url:      url,
		Insecure: c.cfg.InsecureSkipVerify,
		Body:     bodyBytes,
		Headers: map[string]*proto.Header{
			"Content-Type": {
				Values: []string{contentType},
			},
		},
	}

	// Marshal the request
	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, fmt.Errorf("host returned error: %w", err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, fmt.Errorf("failed to unmarshal host response: %w", err)
	}

	// Build the response object
	response := &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
		Header:     make(http.Header),
	}

	// Convert headers if present
	for name, header := range r.Headers {
		response.Header[name] = header.Values
	}

	// Add body if present
	if len(r.Body) > 0 {
		response.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return response, nil
}

func (c *httpClient) Delete(url string) (*Response, error) {
	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "DELETE",
		Url:      url,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}

	// Marshal the request
	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, fmt.Errorf("host returned error: %w", err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, fmt.Errorf("failed to unmarshal host response: %w", err)
	}

	// Build the response object
	response := &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
		Header:     make(http.Header),
	}

	// Convert headers if present
	for name, header := range r.Headers {
		response.Header[name] = header.Values
	}

	// Add body if present
	if len(r.Body) > 0 {
		response.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return response, nil
}

func (c *httpClient) Do(req *Request) (*Response, error) {
	// Read the body content if present
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return &Response{}, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	if req.URL == nil {
		return &Response{}, ErrInvalidURL
	}

	// Create the Protobuf request
	pbReq := &proto.HTTPClient{
		Method:   req.Method,
		Url:      req.URL.String(),
		Insecure: c.cfg.InsecureSkipVerify,
		Body:     bodyBytes,
		Headers:  make(map[string]*proto.Header),
	}

	// Convert headers
	for key, values := range req.Header {
		pbReq.Headers[key] = &proto.Header{
			Values: values,
		}
	}

	// Marshal the request
	b, err := pb.Marshal(pbReq)
	if err != nil {
		return &Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, fmt.Errorf("host returned error: %w", err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, fmt.Errorf("failed to unmarshal host response: %w", err)
	}

	// Build the response object
	response := &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
		Header:     make(http.Header),
	}

	// Convert headers if present
	for name, header := range r.Headers {
		response.Header[name] = header.Values
	}

	// Add body if present
	if len(r.Body) > 0 {
		response.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return response, nil
}

// NewRequest creates a new Request object to use with the Do method
//
// This function provides a way to create custom HTTP requests with
// specific methods, URLs and body content.
func NewRequest(method, urlString string, body io.Reader) (*Request, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	req := &Request{
		Method: method,
		URL:    parsedURL,
		Header: make(http.Header),
	}

	if body != nil {
		req.Body = io.NopCloser(body)
	}

	return req, nil
}
