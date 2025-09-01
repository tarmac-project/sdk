package function

// Function represents a client for invoking other Tarmac functions.
// This interface is currently a stub.
type Function interface{}

// functionClient is a minimal placeholder implementation.
type functionClient struct{}

// Config holds options for the function client.
type Config struct{}

// New creates a new function client.
func New(config *Config) (*functionClient, error) {
    return &functionClient{}, nil
}

// Call invokes a function using the current client configuration.
func (c *functionClient) Call() error {
    return nil
}
