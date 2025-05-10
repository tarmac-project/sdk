/*
Package sdk is the core package for Tarmac SDK v2, providing a modular approach to building WebAssembly functions for Tarmac.

This package provides the central registration point for WebAssembly functions, while all other capabilities
(HTTP, KV, SQL, etc.) are imported as separate modules. This modular design allows you to include only the
functionality your function needs, resulting in smaller WebAssembly modules.

# Basic Usage

Register your function handler with the SDK:

    import (
        "github.com/tarmac-project/sdk"
        "github.com/tarmac-project/sdk/http"
        "github.com/tarmac-project/sdk/log"
    )

    func main() {
        // Register your function handler
        err := sdk.Register(sdk.Config{
            Namespace: "my-service",
            Handler: myHandler,
        })
        if err != nil {
            // handle error
        }
    }

    func myHandler(payload []byte) ([]byte, error) {
        // Create components with same namespace
        logger := log.New(log.Config{Namespace: "my-service"})
        client, _ := http.New(http.Config{Namespace: "my-service"})
        
        logger.Info("Processing request")
        
        // Make HTTP requests
        resp, err := client.Get("https://example.com")
        if err != nil {
            logger.Error("Failed to make request: %v", err)
            return nil, err
        }
        
        // Process response...
        return []byte("Success!"), nil
    }

# Component Packages

The SDK is divided into several component packages:

  - http: HTTP client for making external requests
  - kv: Key-value store for data persistence
  - log: Structured logging capabilities
  - metrics: Metrics reporting and monitoring
  - function: Call other Tarmac functions
  - sql: SQL database integration

Each component provides its own interface and can be imported separately. All components follow
a similar pattern for initialization, typically requiring a namespace at minimum.

# Testing

Each component package includes a mock subpackage to facilitate testing:

    import (
        "github.com/tarmac-project/sdk/http/mock"
    )
    
    func TestMyFunction(t *testing.T) {
        // Create a mock HTTP client
        mockClient := mock.New(mock.Config{
            DefaultResponse: &mock.Response{
                StatusCode: 200,
                Body: []byte(`{"success":true}`),
            },
        })
        
        // Test your function with the mock
        result := myFunctionUnderTest(mockClient)
        
        // Assert on result
    }
*/
package sdk

import (
	"fmt"

	wapc "github.com/wapc/wapc-guest-tinygo"
)

// Config provides configuration options for SDK initialization.
type Config struct {
	// Namespace controls the function namespace to use for host callbacks
	// The default value is "default" which is the global namespace
	Namespace string
	
	// Handler is the function to be registered as the main WebAssembly entry point
	Handler func([]byte) ([]byte, error)
}

// Register registers a function handler with waPC and configures the
// basic SDK settings. This should be called in your main function.
func Register(config Config) error {
	// Validate Handler is not empty
	if config.Handler == nil {
		return fmt.Errorf("function handler cannot be nil")
	}
	
	// Register the provided handler with waPC
	wapc.RegisterFunction("handler", config.Handler)
	
	return nil
}