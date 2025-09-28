package kv

import (
	"errors"
	"fmt"

	"github.com/tarmac-project/protobuf-go/sdk/kvstore"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
	pb "google.golang.org/protobuf/proto"
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

type client struct {
	runtime  sdk.RuntimeConfig
	hostCall func(string, string, string, []byte) ([]byte, error)
}

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

	// ErrHostResponseInvalid indicates that the host returned an invalid or unexpected response.
	ErrHostResponseInvalid = errors.New("host response is invalid or unexpected")

	// ErrHostCall indicates that the waPC host call failed.
	ErrHostCall = errors.New("host call failed")
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
	if key == "" {
		return nil, ErrInvalidKey
	}
	req := &kvstore.KVStoreGet{Key: key}
	b, err := pb.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}
	respBytes, err := c.hostCall(c.runtime.Namespace, "kvstore", "get", b)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHostCall, err)
	}
	var resp kvstore.KVStoreGetResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHostResponseInvalid, err)
	}
	if s := resp.GetStatus(); s != nil && s.Code != 0 {
		if s.Code == 404 {
			return nil, ErrKeyNotFound
		}
		return nil, ErrHostResponseInvalid
	}
	return resp.GetData(), nil
}

// Set stores value under key. It returns ErrInvalidKey or ErrInvalidValue
// for invalid inputs, or wraps host errors.
func (c *client) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	if value == nil {
		return ErrInvalidValue
	}
	req := &kvstore.KVStoreSet{Key: key, Data: value}
	b, err := pb.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal set request: %w", err)
	}
	respBytes, err := c.hostCall(c.runtime.Namespace, "kvstore", "set", b)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHostCall, err)
	}
	var resp kvstore.KVStoreSetResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("%w: %v", ErrHostResponseInvalid, err)
	}
	if s := resp.GetStatus(); s != nil && s.Code != 0 {
		return ErrHostResponseInvalid
	}
	return nil
}

// Delete removes key from the store. Deleting a non-existent key is not an error.
func (c *client) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	req := &kvstore.KVStoreDelete{Key: key}
	b, err := pb.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}
	respBytes, err := c.hostCall(c.runtime.Namespace, "kvstore", "delete", b)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHostCall, err)
	}
	var resp kvstore.KVStoreDeleteResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("%w: %v", ErrHostResponseInvalid, err)
	}
	if s := resp.GetStatus(); s != nil && s.Code != 0 {
		return ErrHostResponseInvalid
	}
	return nil
}

// Keys returns a snapshot of keys currently in the store.
func (c *client) Keys() ([]string, error) {
	req := &kvstore.KVStoreKeys{ReturnProto: true}
	b, err := pb.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys request: %w", err)
	}
	respBytes, err := c.hostCall(c.runtime.Namespace, "kvstore", "keys", b)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHostCall, err)
	}
	var resp kvstore.KVStoreKeysResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHostResponseInvalid, err)
	}
	if s := resp.GetStatus(); s != nil && s.Code != 0 {
		return nil, ErrHostResponseInvalid
	}
	return resp.GetKeys(), nil
}
