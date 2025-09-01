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

type ResponseBuilder struct {
	client *MockClient
	key    string
}

func (r *ResponseBuilder) Return(response *Response) *MockClient {
	// Ensure header is initialized
	if response.Header == nil {
		response.Header = make(http.Header)
	}

	r.client.responses[r.key] = response
	return r.client
}

func (r *ResponseBuilder) ReturnError(err error) *MockClient {
	r.client.responses[r.key] = &Response{
		Error: err,
	}
	return r.client
}
