package httpclient

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
)

// Client provides an interface for making HTTP requests.
type Client interface {
	// Get issues a GET request to the specified URL.
	Get(url string) (*Response, error)

	// Post issues a POST request to the specified URL with the given content type and body.
	Post(url, contentType string, body io.Reader) (*Response, error)

	// Put issues a PUT request to the specified URL with the given content type and body.
	Put(url, contentType string, body io.Reader) (*Response, error)

	// Delete issues a DELETE request to the specified URL.
	Delete(url string) (*Response, error)

	// Do issues a custom HTTP request and returns the response.
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

// HTTPClient implements Client using waPC host calls.
type HTTPClient struct {
	// cfg holds client configuration, including SDKConfig and TLS behavior.
	cfg Config
	// hostCall performs the waPC invocation; tests may override it.
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// Ensure HTTPClient always satisfies the Client interface at compile time.
var _ Client = (*HTTPClient)(nil)

// doHTTPCall marshals the protobuf request, performs the host call, and
// unmarshals the response into a Response using proto getters.
func (c *HTTPClient) doHTTPCall(req *proto.HTTPClient) (*Response, error) {
	b, err := req.MarshalVT()
	if err != nil {
		return &Response{}, errors.Join(ErrMarshalRequest, err)
	}

	resp, err := c.hostCall(c.cfg.SDKConfig.Namespace, "httpclient", "call", b)
	if err != nil {
		return &Response{}, errors.Join(sdk.ErrHostCall, err)
	}

	var r proto.HTTPClientResponse
	if unmarshalErr := r.UnmarshalVT(resp); unmarshalErr != nil {
		return &Response{}, errors.Join(ErrUnmarshalResponse, unmarshalErr)
	}

	status := r.GetStatus()
	if status == nil {
		return &Response{}, sdk.ErrHostResponseInvalid
	}

	statusCode := status.GetCode()
	switch statusCode {
	case hostStatusOK, hostStatusPartial:
		// success path continues
	case hostStatusBadInput, hostStatusMissing, hostStatusError:
		detail := fmt.Sprintf("host status %d", statusCode)
		if msg := status.GetStatus(); msg != "" {
			detail = fmt.Sprintf("%s: %s", detail, msg)
		}
		return &Response{}, errors.Join(sdk.ErrHostError, errors.New(detail))
	default:
		return &Response{}, errors.Join(
			sdk.ErrHostResponseInvalid,
			fmt.Errorf("unexpected host status code %d", statusCode),
		)
	}

	httpCode := int(r.GetCode())
	statusText := http.StatusText(httpCode)

	out := &Response{
		Status:     statusText,
		StatusCode: httpCode,
		Header:     make(http.Header),
	}

	for name, header := range r.GetHeaders() {
		out.Header[name] = header.GetValues()
	}

	if body := r.GetBody(); len(body) > 0 {
		out.Body = io.NopCloser(bytes.NewReader(body))
	}

	return out, nil
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

	// ErrInvalidMethod indicates an HTTP method not permitted by NewRequest.
	ErrInvalidMethod = errors.New("invalid HTTP method")

	// ErrNilRequest indicates Do received a nil Request pointer.
	ErrNilRequest = errors.New("request is nil")
)

const (
	hostStatusOK       = int32(200)
	hostStatusPartial  = int32(206)
	hostStatusBadInput = int32(400)
	hostStatusMissing  = int32(404)
	hostStatusError    = int32(500)
)

// New creates a new HTTP client with the provided configuration.
func New(config Config) (*HTTPClient, error) {
	hc := &HTTPClient{cfg: config}

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
func (c *HTTPClient) Get(urlStr string) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return &Response{}, ErrInvalidURL
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "GET",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}
	return c.doHTTPCall(req)
}

// Post issues a POST to the URL with the provided contentType and body.
func (c *HTTPClient) Post(urlStr, contentType string, body io.Reader) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return &Response{}, ErrInvalidURL
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
	headers := make(map[string]*proto.Header)
	if contentType != "" {
		headers["Content-Type"] = &proto.Header{Values: []string{contentType}}
	}
	req := &proto.HTTPClient{
		Method:   "POST",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Body:     bodyBytes,
		Headers:  headers,
	}
	return c.doHTTPCall(req)
}

// Put issues a PUT to the URL with the provided contentType and body.
func (c *HTTPClient) Put(urlStr, contentType string, body io.Reader) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return &Response{}, ErrInvalidURL
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
	headers := make(map[string]*proto.Header)
	if contentType != "" {
		headers["Content-Type"] = &proto.Header{Values: []string{contentType}}
	}
	req := &proto.HTTPClient{
		Method:   "PUT",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Body:     bodyBytes,
		Headers:  headers,
	}
	return c.doHTTPCall(req)
}

// Delete issues a DELETE to the specified URL.
func (c *HTTPClient) Delete(urlStr string) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || u.Host == "" {
		return &Response{}, ErrInvalidURL
	}

	// Create the Protobuf request
	req := &proto.HTTPClient{
		Method:   "DELETE",
		Url:      urlStr,
		Insecure: c.cfg.InsecureSkipVerify,
		Headers:  make(map[string]*proto.Header),
	}
	return c.doHTTPCall(req)
}

// Do issues a custom request built with NewRequest and returns the response.
func (c *HTTPClient) Do(req *Request) (*Response, error) {
	if req == nil {
		return &Response{}, ErrNilRequest
	}

	// Validate the URL before touching the body stream.
	if req.URL == nil || req.URL.Host == "" {
		return &Response{}, ErrInvalidURL
	}

	// Read the body content if present
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		defer func() { _ = req.Body.Close() }()
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return &Response{}, errors.Join(ErrReadBody, err)
		}
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

	return c.doHTTPCall(pbReq)
}

// NewRequest creates a new Request object to use with the Do method.
//
// This function provides a way to create custom HTTP requests with
// specific methods, URLs and body content.
func NewRequest(method, urlString string, body io.Reader) (*Request, error) {
	// Validate the HTTP method first
	if !isValidMethod(method) {
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

func isValidMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return true
	default:
		return false
	}
}
