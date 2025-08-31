/*
Package kv provides a key-value client for Tarmac functions.

This package allows functions to interact with a key-value store via the Tarmac host
using the waPC protocol. It is designed for WebAssembly environments and uses Protocol
Buffers for payloads.

# Basic Usage

Create a client and perform KV operations:

	kvClient, err := kv.New(kv.Config{Namespace: "my-service"})
	if err != nil {
		// handle error
	}

	// Set a value
	if err := kvClient.Set("foo", []byte("bar")); err != nil {
		// handle error
	}

	// Get a value
	value, err := kvClient.Get("foo")
	if err != nil {
		// handle error
	}

	// Delete a key
	if err := kvClient.Delete("foo"); err != nil {
		// handle error
	}

	// List keys
	keys, err := kvClient.Keys()
	if err != nil {
		// handle error
	}

# Testing with Mocks

Use host-level validation with hostmock, or the in-memory kv/mock client for unit tests:

	import (
		"github.com/tarmac-project/sdk/kv/mock"
	)

	m := mock.New(mock.Config{Seed: map[string][]byte{"a": []byte("1")}})
	_ = m.Set("b", []byte("2"))
	v, _ := m.Get("a")
	keys, _ := m.Keys()
*/
package kv

import (
	"errors"
	"fmt"

	"github.com/tarmac-project/protobuf-go/sdk/kvstore"
	sdk "github.com/tarmac-project/sdk"
	"github.com/wapc/wapc-guest-tinygo"
	pb "google.golang.org/protobuf/proto"
)

type KV interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Keys() ([]string, error)
	Close() error
}

// kvClient implements the KV interface via waPC host calls.
type kvClient struct {
	namespace string
	hostCall  func(string, string, string, []byte) ([]byte, error)
}

// Config provides configuration for the KV client, including namespace and host call function.
type Config struct {
	// Namespace is used for multi-tenant host call scoping. Defaults to "default".
	Namespace string

	// SDKConfig supplies shared SDK-level configuration such as the
	// default Namespace. If Namespace above is set, it takes precedence.
	SDKConfig sdk.RuntimeConfig

	// HostCall is the function to invoke host callbacks. Defaults to wapc.HostCall.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

var (
	ErrInvalidKey          = errors.New("key is invalid")
	ErrInvalidValue        = errors.New("value is invalid")
	ErrKeyNotFound         = errors.New("key not found in store")
	ErrHostResponseInvalid = errors.New("host response is invalid or unexpected")
	ErrHostCall            = errors.New("host call failed")
)

// New returns a KV client configured to communicate with the host via waPC.
func New(config Config) (KV, error) {
	if config.Namespace == "" {
		if config.SDKConfig.Namespace != "" {
			config.Namespace = config.SDKConfig.Namespace
		} else {
			config.Namespace = "default"
		}
	}
	hostFn := config.HostCall
	if hostFn == nil {
		hostFn = wapc.HostCall
	}
	return &kvClient{namespace: config.Namespace, hostCall: hostFn}, nil
}

func (c *kvClient) Close() error {
	return nil
}

func (c *kvClient) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	req := &kvstore.KVStoreGet{Key: key}
	b, err := pb.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}
	respBytes, err := c.hostCall(c.namespace, "kvstore", "get", b)
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

func (c *kvClient) Set(key string, value []byte) error {
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
	respBytes, err := c.hostCall(c.namespace, "kvstore", "set", b)
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

func (c *kvClient) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	req := &kvstore.KVStoreDelete{Key: key}
	b, err := pb.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}
	respBytes, err := c.hostCall(c.namespace, "kvstore", "delete", b)
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

func (c *kvClient) Keys() ([]string, error) {
	req := &kvstore.KVStoreKeys{ReturnProto: true}
	b, err := pb.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys request: %w", err)
	}
	respBytes, err := c.hostCall(c.namespace, "kvstore", "keys", b)
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
