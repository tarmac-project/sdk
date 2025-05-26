package kv

import (
	"errors"
	"fmt"

	"github.com/tarmac-project/protobuf-go/sdk/kvstore"
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

	// HostCall is the function to invoke host callbacks. Defaults to wapc.HostCall.
	HostCall func(string, string, string, []byte) ([]byte, error)
}

var (
	ErrInvalidKey          = errors.New("key is invalid")
	ErrInvalidValue        = errors.New("value is invalid")
	ErrKeyNotFound         = errors.New("key not found in store")
	ErrHostResponseInvalid = errors.New("host response is invalid or unexpected")
)

// New returns a KV client configured to communicate with the host via waPC.
func New(config Config) (KV, error) {
	if config.Namespace == "" {
		config.Namespace = "default"
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
		return nil, fmt.Errorf("host returned error: %w", err)
	}
	var resp kvstore.KVStoreGetResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host response: %w", err)
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
		return fmt.Errorf("host returned error: %w", err)
	}
	var resp kvstore.KVStoreSetResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal host response: %w", err)
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
		return fmt.Errorf("host returned error: %w", err)
	}
	var resp kvstore.KVStoreDeleteResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal host response: %w", err)
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
		return nil, fmt.Errorf("host returned error: %w", err)
	}
	var resp kvstore.KVStoreKeysResponse
	if err := pb.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host response: %w", err)
	}
	return resp.GetKeys(), nil
}
