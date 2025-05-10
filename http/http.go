package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	pb "google.golang.org/protobuf/proto"
)

type Client interface {
	Get(url string) (*Response, error)
	Post(url, contentType string, body io.Reader) (*Response, error)
	Put(url, contentType string, body io.Reader) (*Response, error)
	Delete(url string) (*Response, error)
	Do(req *Request) (*Response, error)
}

type Config struct {
	Namespace          string
	InsecureSkipVerify bool
	HostCall           func(string, string, string, []byte) ([]byte, error)
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

func New(config Config) (Client, error) {
	return &httpClient{hostCall: config.HostCall, cfg: config}, nil
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

	resp, err := c.hostCall(c.cfg.Namespace, "http", "http", b)
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
	resp, err := c.hostCall(c.cfg.Namespace, "http", "http", b)
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
	resp, err := c.hostCall(c.cfg.Namespace, "http", "http", b)
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
	resp, err := c.hostCall(c.cfg.Namespace, "http", "http", b)
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
	resp, err := c.hostCall(c.cfg.Namespace, "http", "http", b)
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
