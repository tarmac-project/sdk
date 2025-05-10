# Tarmac SDK v2

This is the official SDK for developing WebAssembly functions for [Tarmac](https://github.com/tarmac-project/tarmac).

## Features

- **Modular Design**: Import only the components you need
- **Simplified Mocking**: Each component has its own mock package for testing
- **Protocol Buffer Based**: Efficient binary serialization for host communication
- **WebAssembly First**: Built specifically for the WebAssembly environment

## Components

The SDK is divided into several modules, each with its own purpose:

- **sdk**: Core package for function registration
- **http**: HTTP client for making external requests
- **kv**: Key-value store for data persistence
- **log**: Structured logging capabilities
- **metrics**: Metrics reporting and monitoring
- **function**: Call other Tarmac functions
- **sql**: SQL database integration

## Getting Started

To use the SDK in your project, import the components you need:

```go
import (
    "github.com/tarmac-project/sdk"
    "github.com/tarmac-project/sdk/http"
    "github.com/tarmac-project/sdk/log"
)

func main() {
    // Register your function handler
    sdk.Register(sdk.Config{
        Namespace: "my-service",
        Handler: myHandler,
    })
}

func myHandler(payload []byte) ([]byte, error) {
    // Create components with same namespace
    logger := log.New(log.Config{Namespace: "my-service"})
    client := http.New(http.Config{Namespace: "my-service"})
    
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
```

## Testing with Mocks

Each component package includes a `mock` subpackage for testing:

```go
import (
    "github.com/tarmac-project/sdk/http/mock"
)

func TestMyFunction(t *testing.T) {
    // Create a mock HTTP client
    mockClient := mock.New(mock.Config{
        DefaultResponse: &mock.Response{
            StatusCode: 200,
            Status: "OK",
            Body: []byte(`{"success":true}`),
        },
    })
    
    // Configure specific endpoint responses
    mockClient.On("GET", "https://example.com").Return(&mock.Response{
        StatusCode: 200,
        Status: "OK",
        Body: []byte(`{"data":"example"}`),
    })
    
    // Call your function with the mock client
    result := myFunctionUnderTest(mockClient)
    
    // Assert on the result
    // ...
}
```

## Building

To build and test the SDK, use the provided Makefiles:

```bash
# Build all components
make build

# Run tests for all components
make test

# Format code
make format

# Lint code
make lint
```

## License

This project is licensed under the [MIT License](LICENSE).