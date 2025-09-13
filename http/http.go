package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
	pb "google.golang.org/protobuf/proto"
)

// validMethods lists HTTP methods accepted by NewRequest.
var validMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// Client provides an interface for making HTTP requests.
type Client interface {
	Get(url string) (*Response, error)
	Post(url, contentType string, body io.Reader) (*Response, error)
	Put(url, contentType string, body io.Reader) (*Response, error)
	Delete(url string) (*Response, error)
	Do(req *Request) (*Response, error)
}

// Config configures the HTTP client behavior and host integration.
//
// SDKConfig supplies the namespace used when making waPC host calls. If the
// Namespace is empty, it defaults to sdk.DefaultNamespace during New.
// InsecureSkipVerify controls TLS verification behavior on the host side when
// supported by the runtime. HostCall allows tests to inject a custom host
// function; when nil, the client uses wapc.HostCall.
type Config struct {
	// SDKConfig provides the runtime namespace for host calls.
	SDKConfig sdk.RuntimeConfig
	// InsecureSkipVerify disables TLS verification when supported.
	InsecureSkipVerify bool
	// HostCall overrides the waPC host function used for requests.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

// httpClient implements Client using waPC host calls.
type httpClient struct {
	// cfg holds client configuration, including SDKConfig and TLS behavior.
	cfg Config
	// hostCall performs the waPC invocation; tests may override it.
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// Response represents an HTTP response returned by the host.
type Response struct {
	// Status is the HTTP status text (e.g., "OK").
	Status string
	// StatusCode is the numeric HTTP status code (e.g., 200).
	StatusCode int
	// Header contains response headers. Nil is treated as empty.
	Header http.Header
	// Body is the response payload stream. It may be nil for empty bodies.
	Body io.ReadCloser
}

// Request represents an HTTP request to be sent by the client.
type Request struct {
	// Method is the HTTP method (e.g., GET, POST).
	Method string
	// URL is the full request URL; Host must be non-empty.
	URL *url.URL
	// Header holds request headers. Nil is treated as empty.
	Header http.Header
	// Body is an optional request body stream.
	Body io.ReadCloser
}

var (
	// ErrInvalidURL indicates a malformed or unsupported URL.
	ErrInvalidURL = errors.New("invalid URL provided")
	// ErrMarshalRequest wraps failures while encoding the request payload.
	ErrMarshalRequest = errors.New("failed to create request")
	// ErrReadBody wraps failures while reading a request body stream.
	ErrReadBody = errors.New("failed to read request body")
	// ErrUnmarshalResponse wraps failures while decoding the host response.
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")
	// ErrHostCall wraps errors returned from the waPC host call.
	ErrHostCall = errors.New("host call failed")
	// ErrInvalidMethod indicates an HTTP method not permitted by NewRequest.
	ErrInvalidMethod = errors.New("invalid HTTP method")
)

// New creates a new HTTP client with the provided configuration.
func New(config Config) (Client, error) {
	hc := &httpClient{cfg: config}

	// Set default namespace if not provided
	if hc.cfg.SDKConfig.Namespace == "" {
		hc.cfg.SDKConfig.Namespace = sdk.DefaultNamespace
	}

	// Set HostCall function if provided
	hc.hostCall = wapc.HostCall
	if config.HostCall != nil {
		hc.hostCall = config.HostCall
	}

	return hc, nil
}

// Get issues a GET to the specified URL and returns the response.
func (c *httpClient) Get(urlStr string) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return nil, ErrInvalidURL
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "GET",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}

	// Marshal the request
	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	// Call the host
	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(ErrHostCall, err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, err)
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

// Post issues a POST to the URL with the provided contentType and body.
func (c *httpClient) Post(urlStr, contentType string, body io.Reader) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return nil, ErrInvalidURL
	}

	// Read the body content if present
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return &Response{}, errors.Join(ErrReadBody, err)
		}
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "POST",
		Url:      urlStr,
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
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(ErrHostCall, err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, err)
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

// Put issues a PUT to the URL with the provided contentType and body.
func (c *httpClient) Put(urlStr, contentType string, body io.Reader) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return nil, ErrInvalidURL
	}

	// Read the body content if present
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return &Response{}, errors.Join(ErrReadBody, err)
		}
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "PUT",
		Url:      urlStr,
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
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(ErrHostCall, err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, err)
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

// Delete issues a DELETE to the specified URL.
func (c *httpClient) Delete(urlStr string) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrInvalidURL
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "DELETE",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}

	// Marshal the request
	b, err := pb.Marshal(req)
	if err != nil {
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(ErrHostCall, err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, err)
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

// Do issues a custom request built with NewRequest and returns the response.
func (c *httpClient) Do(req *Request) (*Response, error) {
	// Read the body content if present
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return &Response{}, errors.Join(ErrReadBody, err)
		}
	}

	// Validate the URL
	if req.URL == nil || req.URL.Host == "" {
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
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	// Make the host call
	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(ErrHostCall, err)
	}

	// Unmarshal the response
	var r proto.HTTPClientResponse
	if err := pb.Unmarshal(resp, &r); err != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, err)
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

// NewRequest creates a new Request object to use with the Do method.
//
// This function provides a way to create custom HTTP requests with
// specific methods, URLs and body content.
func NewRequest(method, urlString string, body io.Reader) (*Request, error) {
	// Validate the HTTP method first
	if _, ok := validMethods[method]; !ok {
		return nil, ErrInvalidMethod
	}

	// Validate the URL
	parsedURL, err := url.Parse(urlString)
	if err != nil || parsedURL == nil || parsedURL.Host == "" {
		return nil, ErrInvalidURL
	}

	// Create the Request object
	req := &Request{
		Method: method,
		URL:    parsedURL,
		Header: make(http.Header),
	}

	// Set the body if provided
	if body != nil {
		req.Body = io.NopCloser(body)
	}

	return req, nil
}
