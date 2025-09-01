package http

import (
	"strings"
	"testing"

	sdkproto "github.com/tarmac-project/protobuf-go/sdk"
	proto "github.com/tarmac-project/protobuf-go/sdk/http"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
	pb "google.golang.org/protobuf/proto"
)

func BenchmarkHTTPClient(b *testing.B) {
	// Create a mock response generator
	createResponseFunc := func() []byte {
		resp := &proto.HTTPClientResponse{
			Status: &sdkproto.Status{
				Status: "OK",
				Code:   200,
			},
			Headers: map[string]*proto.Header{
				"Content-Type": {
					Values: []string{"application/json"},
				},
			},
			Body: []byte(`{"message":"success"}`),
		}

		respBytes, _ := pb.Marshal(resp)
		return respBytes
	}

	// Configure the mock
	mock, err := hostmock.New(hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: "httpclient",
		ExpectedFunction:   "call",
		Response:           createResponseFunc,
	})

	if err != nil {
		b.Fatalf("Failed to create mock: %v", err)
	}

	// Create the client
	client, err := New(Config{
		SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"},
		HostCall:  mock.HostCall,
	})

	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	b.Run("GET", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Get("http://example.com")
			if err != nil {
				b.Fatalf("Failed to make GET request: %v", err)
			}
		}
	})

	b.Run("POST", func(b *testing.B) {
		data := strings.NewReader(`{"data":"test"}`)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Reset the reader position for each iteration
			data.Reset(`{"data":"test"}`)
			_, err := client.Post("http://example.com", "application/json", data)
			if err != nil {
				b.Fatalf("Failed to make POST request: %v", err)
			}
		}
	})
}
