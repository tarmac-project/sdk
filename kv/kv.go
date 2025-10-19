package kv

import (
	"errors"
	"fmt"

	kvstore "github.com/tarmac-project/protobuf-go/sdk/kvstore"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

// Client represents a key-value capability client.
type Client interface {
	// Config returns the runtime configuration used by the client.
	Config() sdk.RuntimeConfig

	// Get returns the value for key or an error. If the key is not found,
	// ErrKeyNotFound is returned.
	Get(key string) ([]byte, error)

	// Set stores value under key. It returns an error for invalid inputs
	// or host call failures.
	Set(key string, value []byte) error

	// Delete removes key. Deleting a non-existent key does not error.
	Delete(key string) error

	// Keys returns a snapshot of keys in the store.
	Keys() ([]string, error)

	// Close releases resources held by the client.
	Close() error
}

// Config controls construction of a key-value client.
type Config struct {
	// SDKConfig provides the runtime namespace for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for requests.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

// client implements Client using a configured waPC host call.
type client struct {
	// runtime carries the namespace and other shared configuration for host calls.
	runtime sdk.RuntimeConfig

	// hostCall issues waPC invocations on behalf of the client.
	hostCall func(string, string, string, []byte) ([]byte, error)
}

// Ensure client implements the Client interface at compile time.
var _ Client = (*client)(nil)

// Config returns a copy of the runtime configuration.
func (c *client) Config() sdk.RuntimeConfig {
	return c.runtime
}

var (
	// ErrInvalidKey indicates that the provided key is empty or otherwise invalid.
	ErrInvalidKey = errors.New("key is invalid")

	// ErrInvalidValue indicates that the provided value is nil or invalid.
	ErrInvalidValue = errors.New("value is invalid")

	// ErrKeyNotFound indicates that the requested key does not exist.
	ErrKeyNotFound = errors.New("key not found in store")
)

const (
	// statusOK indicates a successful operation.
	statusOK = int32(200)

	// statusNotFound indicates that the requested key does not exist.
	statusNotFound = int32(404)

	// statusError indicates that an error occurred during the operation.
	statusError = int32(500)
)

// New creates a new key-value client.
func New(config Config) (Client, error) {
	runtime := config.SDKConfig
	if runtime.Namespace == "" {
		runtime.Namespace = sdk.DefaultNamespace
	}

	hostCall := config.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &client{
		runtime:  runtime,
		hostCall: hostCall,
	}, nil
}

// Close releases resources associated with the client. It is a no-op.
func (c *client) Close() error {
	return nil
}

// Get retrieves the value for key or returns ErrKeyNotFound if missing.
func (c *client) Get(key string) ([]byte, error) {
	// Validate provided key
	if key == "" {
		return nil, ErrInvalidKey
	}

	// Construct and marshal the get request
	req := &kvstore.KVStoreGet{Key: key}
	b, err := req.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	// Issue the host call and always inspect the payload.
	respBytes, callErr := c.hostCall(c.runtime.Namespace, "kvstore", "get", b)
	if callErr != nil && len(respBytes) == 0 {
		return nil, errors.Join(sdk.ErrHostCall, callErr)
	}

	// Attempt to unmarshal whatever the host returned.
	var resp kvstore.KVStoreGetResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return nil, errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid, unmarshalErr)
		}
		return nil, errors.Join(sdk.ErrHostResponseInvalid, unmarshalErr)
	}

	status := resp.GetStatus()
	if status != nil && status.GetCode() == statusOK {
		return resp.GetData(), nil
	}

	if status != nil && status.GetCode() == statusNotFound {
		return nil, ErrKeyNotFound
	}

	if status != nil && status.GetCode() == statusError {
		if callErr != nil {
			return nil, errors.Join(sdk.ErrHostError, callErr)
		}
		return nil, sdk.ErrHostError
	}

	return nil, sdk.ErrHostResponseInvalid
}

// Set stores value under key. It returns ErrInvalidKey or ErrInvalidValue
// for invalid inputs, or wraps host errors.
func (c *client) Set(key string, value []byte) error {
	// Validate inputs
	if key == "" {
		return ErrInvalidKey
	}

	if len(value) == 0 {
		return ErrInvalidValue
	}

	// Construct and marshal the set request
	req := &kvstore.KVStoreSet{Key: key, Data: value}
	b, err := req.MarshalVT()
	if err != nil {
		return fmt.Errorf("failed to marshal set request: %w", err)
	}

	// Issue the host call and inspect the payload even on error
	respBytes, callErr := c.hostCall(c.runtime.Namespace, "kvstore", "set", b)
	if callErr != nil && (len(respBytes) == 0) {
		return errors.Join(sdk.ErrHostCall, callErr)
	}

	// Unmarshal the response from the host
	var resp kvstore.KVStoreSetResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid, unmarshalErr)
		}
		return errors.Join(sdk.ErrHostResponseInvalid, unmarshalErr)
	}

	status := resp.GetStatus()
	if status != nil && status.GetCode() == statusOK {
		return nil
	}

	if status != nil && status.GetCode() == statusError {
		if callErr != nil {
			return errors.Join(sdk.ErrHostError, callErr)
		}
		return sdk.ErrHostError
	}

	return sdk.ErrHostResponseInvalid
}

// Delete removes key from the store. Deleting a non-existent key is not an error.
func (c *client) Delete(key string) error {
	// Validate key input up front to avoid unnecessary host calls.
	if key == "" {
		return ErrInvalidKey
	}

	// Marshal the delete request for the host capability.
	req := &kvstore.KVStoreDelete{Key: key}
	b, err := req.MarshalVT()
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	// Invoke the host; keep the bytes for status parsing even when an error is returned.
	respBytes, callErr := c.hostCall(c.runtime.Namespace, "kvstore", "delete", b)
	if callErr != nil && len(respBytes) == 0 {
		return errors.Join(sdk.ErrHostCall, callErr)
	}

	// Decode the payload; surface both host and decoding errors when applicable.
	var resp kvstore.KVStoreDeleteResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid, unmarshalErr)
		}
		return errors.Join(sdk.ErrHostResponseInvalid, unmarshalErr)
	}

	status := resp.GetStatus()
	if status != nil && (status.GetCode() == statusOK || status.GetCode() == statusNotFound) {
		return nil
	}

	if status != nil && status.GetCode() == statusError {
		if callErr != nil {
			return errors.Join(sdk.ErrHostError, callErr)
		}
		return sdk.ErrHostError
	}

	return sdk.ErrHostResponseInvalid
}

// Keys returns a snapshot of keys currently in the store.
func (c *client) Keys() ([]string, error) {
	// Build a request that asks the host to return a protobuf-encoded key list.
	req := &kvstore.KVStoreKeys{ReturnProto: true}
	b, err := req.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys request: %w", err)
	}

	// Execute the host call; retain bytes even when the host reports an error.
	respBytes, callErr := c.hostCall(c.runtime.Namespace, "kvstore", "keys", b)
	if callErr != nil && len(respBytes) == 0 {
		return nil, errors.Join(sdk.ErrHostCall, callErr)
	}

	// Decode the protobuf payload and combine errors if both occur.
	var resp kvstore.KVStoreKeysResponse
	if unmarshalErr := resp.UnmarshalVT(respBytes); unmarshalErr != nil {
		if callErr != nil {
			return nil, errors.Join(sdk.ErrHostCall, callErr, sdk.ErrHostResponseInvalid, unmarshalErr)
		}
		return nil, errors.Join(sdk.ErrHostResponseInvalid, unmarshalErr)
	}

	status := resp.GetStatus()
	if status != nil && status.GetCode() == statusOK {
		return resp.GetKeys(), nil
	}

	if status != nil && status.GetCode() == statusError {
		if callErr != nil {
			return nil, errors.Join(sdk.ErrHostError, callErr)
		}
		return nil, sdk.ErrHostError
	}

	return nil, sdk.ErrHostResponseInvalid
}
