package mock

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	sdk "github.com/tarmac-project/sdk/http"
)

type MockClient struct {
	// responses maps method+url to predefined responses
	responses map[string]*Response

	// DefaultResponse is returned when no matching response is found
	DefaultResponse *Response

	// Calls tracks which endpoints were called
	Calls []Call
}

type Response struct {
	StatusCode int
	Status     string
	Body       []byte
	Header     http.Header
	Error      error
}

type Call struct {
	Method string
	URL    string
	Body   []byte
	Header http.Header
}

type Config struct {
	DefaultResponse *Response
}

// New creates a new mock HTTP client.
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

// responseFor returns the configured response for method+url or the default.
func (m *MockClient) responseFor(method, url string) *Response {
	key := method + " " + url
	if resp, ok := m.responses[key]; ok {
		return resp
	}
	return m.DefaultResponse
}

// toSDKResponse converts a mock Response into an sdk.Response with copied headers.
func toSDKResponse(r *Response) *sdk.Response {
	resp := &sdk.Response{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(r.Body))),
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

func (m *MockClient) On(method, url string) *ResponseBuilder {
	key := method + " " + url
	return &ResponseBuilder{
		client: m,
		key:    key,
	}
}

func (m *MockClient) Get(url string) (*sdk.Response, error) {
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

func (m *MockClient) Post(url, contentType string, body io.Reader) (*sdk.Response, error) {
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

func (m *MockClient) Put(url, contentType string, body io.Reader) (*sdk.Response, error) {
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

func (m *MockClient) Delete(url string) (*sdk.Response, error) {
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

func (m *MockClient) Do(req *sdk.Request) (*sdk.Response, error) {
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
