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

type Client interface {
	Get(url string) (*Response, error)
	Post(url, contentType string, body io.Reader) (*Response, error)
	Put(url, contentType string, body io.Reader) (*Response, error)
	Delete(url string) (*Response, error)
	Do(req *Request) (*Response, error)
}

type Config struct {
	SDKConfig sdk.RuntimeConfig
	InsecureSkipVerify bool
	HostCall func(string, string, string, []byte) ([]byte, error)
}

type httpClient struct {
	cfg      Config
	hostCall func(string, string, string, []byte) ([]byte, error)
}

type Response struct {
	Status     string
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   io.ReadCloser
}

var (
	ErrInvalidURL = errors.New("invalid URL provided")
	ErrMarshalRequest = errors.New("failed to create request")
	ErrReadBody = errors.New("failed to read request body")
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")
	ErrHostCall = errors.New("host call failed")
	ErrInvalidMethod = errors.New("invalid HTTP method")
)

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

func (c *httpClient) Put(urlStr, contentType string, body io.Reader) (*Response, error) {
	// Validate the URL
	u, err := url.Parse(urlStr)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
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

// NewRequest creates a new Request object to use with the Do method
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
