package kv

import (
	"errors"
)

type KV interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Keys() ([]string, error)
	Close() error
}

type kvClient struct{}

type Config struct{}

var (
	ErrInvalidKey   = errors.New("key is invalid")
	ErrInvalidValue = errors.New("value is invalid")
)

func New(config Config) (KV, error) {
	return &kvClient{}, nil
}

func (c *kvClient) Close() error {
	return nil
}

func (c *kvClient) Get(key string) ([]byte, error) {
	return nil, nil
}

func (c *kvClient) Set(key string, value []byte) error {
	return nil
}

func (c *kvClient) Delete(key string) error {
	return nil
}

func (c *kvClient) Keys() ([]string, error) {
	return nil, nil
}
