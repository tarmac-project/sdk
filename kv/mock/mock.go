/*
Package mock provides a mock implementation of the kv.KV interface for testing Tarmac functions.

This package implements an in-memory key/value store with configurable behaviors so you can test
code that depends on the kv component without invoking host calls. You can pre-seed data, override
per-operation behavior, and record calls for assertions.

# Basic Usage

Create a mock client with optional seed data:

	import (
		"testing"

		"github.com/tarmac-project/sdk/kv/mock"
	)

	func TestSomething(t *testing.T) {
		m := mock.New(mock.Config{Seed: map[string][]byte{"a": []byte("1")}})
		v, err := m.Get("a")
		// assert v == "1" and err == nil
	}

# Overriding Behavior

Override responses per operation/key using a fluent builder:

	m.OnGet("missing").ReturnValue(nil).ReturnError(kv.ErrKeyNotFound)
	m.OnSet("bad").ReturnError(fmt.Errorf("reject set"))
	m.OnDelete("ghost").ReturnError(kv.ErrKeyNotFound)
	m.OnKeys().ReturnKeys([]string{"x","y"})

# Inspecting Calls

	for _, c := range m.Calls {
		// c.Op, c.Key, c.Value
	}
*/
package mock

import (
	"fmt"
	"sort"

	sdk "github.com/tarmac-project/sdk/kv"
)

// Operation names used for per-call configuration.
const (
	opGet    = "GET"
	opSet    = "SET"
	opDelete = "DELETE"
	opKeys   = "KEYS"
)

// Config configures the mock client.
type Config struct {
	// Seed pre-populates the in-memory store.
	Seed map[string][]byte
}

// Response describes a configured mock outcome.
type Response struct {
	// Value applies to GET and stores the supplied bytes.
	Value []byte
	// Keys applies to KEYS.
	Keys []string
	// Err indicates an error to return for the operation.
	Err error
	// storeOnSet controls whether SET updates the in-memory store when a
	// configured SET response exists and Err == nil. Defaults to true.
	storeOnSet *bool
}

// ResponseBuilder allows fluent configuration of responses.
type ResponseBuilder struct {
	m   *Client
	key string // composite key: OP + " " + target
}

// ReturnValue sets bytes returned by GET; also used for SET default value passthrough.
func (b *ResponseBuilder) ReturnValue(v []byte) *ResponseBuilder {
	r := b.m.getOrCreate(b.key)
	r.Value = v
	b.m.responses[b.key] = r
	return b
}

// ReturnKeys sets keys returned by KEYS.
func (b *ResponseBuilder) ReturnKeys(keys []string) *ResponseBuilder {
	r := b.m.getOrCreate(b.key)
	r.Keys = append([]string(nil), keys...)
	b.m.responses[b.key] = r
	return b
}

// ReturnError sets an error for the configured operation.
func (b *ResponseBuilder) ReturnError(err error) *Client {
	r := b.m.getOrCreate(b.key)
	r.Err = err
	b.m.responses[b.key] = r
	return b.m
}

// StoreOnSet controls whether a configured SET without error updates the store (default true).
func (b *ResponseBuilder) StoreOnSet(v bool) *ResponseBuilder {
	r := b.m.getOrCreate(b.key)
	r.storeOnSet = &v
	b.m.responses[b.key] = r
	return b
}

// Call records an operation performed against the mock.
type Call struct {
	Op    string
	Key   string
	Value []byte
}

// Client implements sdk.KV for tests.
type Client struct {
	store     map[string][]byte
	responses map[string]Response
	// Calls stores a history of operations for assertions.
	Calls []Call
}

// New creates a new mock KV client.
func New(cfg Config) *Client {
	st := make(map[string][]byte)
	for k, v := range cfg.Seed {
		st[k] = append([]byte(nil), v...)
	}
	return &Client{
		store:     st,
		responses: make(map[string]Response),
		Calls:     []Call{},
	}
}

// OnGet configures a GET response for a key.
func (m *Client) OnGet(key string) *ResponseBuilder {
	return &ResponseBuilder{m: m, key: opGet + " " + key}
}

// OnSet configures a SET response for a key.
func (m *Client) OnSet(key string) *ResponseBuilder {
	return &ResponseBuilder{m: m, key: opSet + " " + key}
}

// OnDelete configures a DELETE response for a key.
func (m *Client) OnDelete(key string) *ResponseBuilder {
	return &ResponseBuilder{m: m, key: opDelete + " " + key}
}

// OnKeys configures the KEYS response.
func (m *Client) OnKeys() *ResponseBuilder { return &ResponseBuilder{m: m, key: opKeys} }

// getOrCreate returns an existing response config or a new one.
func (m *Client) getOrCreate(k string) Response {
	if r, ok := m.responses[k]; ok {
		return r
	}
	return Response{}
}

// ensureStoreOnSet returns the effective storeOnSet flag for a response.
func ensureStoreOnSet(r Response) bool {
	if r.storeOnSet == nil {
		return true
	}
	return *r.storeOnSet
}

// Get implements sdk.KV.
func (m *Client) Get(key string) ([]byte, error) {
	m.Calls = append(m.Calls, Call{Op: opGet, Key: key})
	if key == "" {
		return nil, sdk.ErrInvalidKey
	}
	if r, ok := m.responses[opGet+" "+key]; ok {
		return r.Value, r.Err
	}
	v, ok := m.store[key]
	if !ok {
		return nil, sdk.ErrKeyNotFound
	}
	return append([]byte(nil), v...), nil
}

// Set implements sdk.KV.
func (m *Client) Set(key string, value []byte) error {
	m.Calls = append(m.Calls, Call{Op: opSet, Key: key, Value: append([]byte(nil), value...)})
	if key == "" {
		return sdk.ErrInvalidKey
	}
	if value == nil {
		return sdk.ErrInvalidValue
	}
	if r, ok := m.responses[opSet+" "+key]; ok {
		if r.Err != nil {
			return r.Err
		}
		if ensureStoreOnSet(r) {
			m.store[key] = append([]byte(nil), value...)
		}
		return nil
	}
	m.store[key] = append([]byte(nil), value...)
	return nil
}

// Delete implements sdk.KV.
func (m *Client) Delete(key string) error {
	m.Calls = append(m.Calls, Call{Op: opDelete, Key: key})
	if key == "" {
		return sdk.ErrInvalidKey
	}
	if r, ok := m.responses[opDelete+" "+key]; ok {
		return r.Err
	}
	if _, ok := m.store[key]; !ok {
		return sdk.ErrKeyNotFound
	}
	delete(m.store, key)
	return nil
}

// Keys implements sdk.KV.
func (m *Client) Keys() ([]string, error) {
	m.Calls = append(m.Calls, Call{Op: opKeys})
	if r, ok := m.responses[opKeys]; ok {
		return append([]string(nil), r.Keys...), r.Err
	}
	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// Close implements sdk.KV.
func (m *Client) Close() error { return nil }

// Example errors used in tests of this mock. Exported for convenience.
var (
	// ErrExample is a sentinel error to help tests customize failures.
	ErrExample = fmt.Errorf("kv mock example error")
)
