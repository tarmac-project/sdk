package httpclient

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"strconv"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
)

// okBenchResponse returns a small, valid protobuf response for happy-path benches.
func okBenchResponse() []byte {
	resp := &proto.HTTPClientResponse{
		Status:  &sdkproto.Status{Status: "OK", Code: 200},
		Headers: map[string]*proto.Header{"Content-Type": {Values: []string{"application/json"}}},
		Body:    []byte(`{"message":"success"}`),
	}
	b, _ := resp.MarshalVT()
	return b
}

func BenchmarkHTTPClient(b *testing.B) {
	// Build a client with a fast, happy-path hostmock.
	mock, err := hostmock.New(hostmock.Config{
		ExpectedNamespace:  sdk.DefaultNamespace,
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
		Response:           okBenchResponse,
	})
	if err != nil {
		b.Fatalf("hostmock: %v", err)
	}
	c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: sdk.DefaultNamespace}, HostCall: mock.HostCall})
	if err != nil {
		b.Fatalf("client: %v", err)
	}

	// Shortcuts: happy path GET/POST/PUT/DELETE
	b.Run("Shortcuts", func(b *testing.B) {
		b.ReportAllocs()

		tt := []struct {
			name        string
			method      string
			url         string
			contentType string
			body        string
		}{
			{"GET", "GET", "http://example.com", "", ""},
			{"POST", "POST", "http://example.com", "application/json", `{"data":"test"}`},
			{"PUT", "PUT", "http://example.com/1", "application/json", `{"x":1}`},
			{"DELETE", "DELETE", "http://example.com/1", "", ""},
		}

		for _, tc := range tt {
			b.Run(tc.name, func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					var (
						r     *Response
						opErr error
					)
					switch tc.method {
					case "GET":
						r, opErr = c.Get(tc.url)
					case "POST":
						rd := strings.NewReader(tc.body)
						r, opErr = c.Post(tc.url, tc.contentType, rd)
					case "PUT":
						rd := strings.NewReader(tc.body)
						r, opErr = c.Put(tc.url, tc.contentType, rd)
					case "DELETE":
						r, opErr = c.Delete(tc.url)
					}
					if opErr != nil {
						b.Fatalf("%s failed: %v", tc.name, opErr)
					}
					if r.Body != nil {
						io.Copy(io.Discard, r.Body)
						r.Body.Close()
					}
				}
			})
		}
	})

	// Do: exercise multiple verbs with varying payload sizes.
	b.Run("Do", func(b *testing.B) {
		b.ReportAllocs()

		// Prebuild payloads to avoid reallocating in the hot loop.
		small := []byte(`{"data":"test"}`)
		medium := bytes.Repeat([]byte("b"), 8*1024) // ~8KiB
		large := bytes.Repeat([]byte("a"), 64*1024) // ~64KiB

		// Helper: build deterministic header key list once per case.
		buildHeaderKeys := func(n int) []string {
			if n <= 0 {
				return nil
			}
			keys := make([]string, n)
			for i := range n {
				keys[i] = "X-Bench-H-" + strconv.Itoa(i)
			}
			return keys
		}

		tt := []struct {
			name        string
			method      string
			url         string
			contentType string
			payload     []byte
			headerCount int // number of additional headers to set on request
		}{
			{"GET/no-body", "GET", "http://example.com", "", nil, 0},
			{"GET/no-body/50hdrs", "GET", "http://example.com", "", nil, 50},
			{"HEAD/no-body", "HEAD", "http://example.com", "", nil, 0},
			{"OPTIONS/no-body", "OPTIONS", "http://example.com", "", nil, 0},
			{"POST/small", "POST", "http://example.com", "application/json", small, 0},
			{"POST/medium/25hdrs", "POST", "http://example.com", "application/json", medium, 25},
			{"POST/large/100hdrs", "POST", "http://example.com", "application/json", large, 100},
			{"PUT/small", "PUT", "http://example.com/1", "application/json", small, 0},
			{"PATCH/small/25hdrs", "PATCH", "http://example.com/1", "application/json", small, 25},
			{"DELETE/no-body", "DELETE", "http://example.com/1", "", nil, 0},
		}

		for _, tc := range tt {
			b.Run(tc.name, func(b *testing.B) {
				hdrKeys := buildHeaderKeys(tc.headerCount)
				b.ResetTimer()
				for range b.N {
					var body io.Reader
					if tc.payload != nil {
						body = bytes.NewReader(tc.payload)
					}
					req, reqErr := NewRequest(tc.method, tc.url, body)
					if reqErr != nil {
						b.Fatalf("new request: %v", reqErr)
					}
					if tc.contentType != "" {
						req.Header.Set("Content-Type", tc.contentType)
					}
					// Add additional headers if requested.
					for _, k := range hdrKeys {
						req.Header.Set(k, "v")
					}
					r, runErr := c.Do(req)
					if runErr != nil {
						b.Fatalf("%s failed: %v", tc.name, runErr)
					}
					if r.Body != nil {
						io.Copy(io.Discard, r.Body)
						r.Body.Close()
					}
				}
			})
		}
	})
}
