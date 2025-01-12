package http

import (
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

	return &Response{
		Status:     r.Status.Status,
		StatusCode: int(r.Status.Code),
	}, nil
}

func (c *httpClient) Post(url, contentType string, body io.Reader) (*Response, error) {
	return &Response{}, nil
}

func (c *httpClient) Put(url, contentType string, body io.Reader) (*Response, error) {
	return &Response{}, nil
}

func (c *httpClient) Delete(url string) (*Response, error) {
	return &Response{}, nil
}

func (c *httpClient) Do(req *Request) (*Response, error) {
	return &Response{}, nil
}

func NewRequest(method, url string, body io.Reader) (*Request, error) {
	return &Request{}, nil
}
