/*
Package hostmock provides a friendly pretend host for waPC calls.

It’s designed primarily for SDK development and advanced tests where you want
to validate exactly what a component is sending to the Tarmac host—without
needing a real host running. No real hosts were harmed in the making of these tests.

Why use hostmock?

  - Validate routing: ensure calls use the expected namespace, capability, and function.
  - Inspect payloads: plug in a PayloadValidator to assert protobuf contents.
  - Script responses: return custom bytes or simulate failures.

When should I use it?

Most users writing functions should prefer component-level mocks:

  - github.com/tarmac-project/sdk/http/mock
  - github.com/tarmac-project/sdk/kv/mock

They’re simpler and avoid coupling your tests to the wire format. Reach for
hostmock when you need to assert the waPC payloads or validate capability routing.

Quick start

	m, _ := hostmock.New(hostmock.Config{
	  ExpectedNamespace:  "tarmac",
	  ExpectedCapability: "httpclient",
	  ExpectedFunction:   "call",
	  PayloadValidator: func(p []byte) error {
	    // Unmarshal and assert fields here
	    return nil
	  },
	  Response: func() []byte { return []byte("ok") },
	})

	// Inject into a component under test
	resp, err := m.HostCall("tarmac", "httpclient", "call", []byte("payload"))

Behavior

  - If Fail is true and Error is set, HostCall returns that error.
  - If Fail is true and Error is nil, HostCall returns ErrOperationFailed.
  - Otherwise, HostCall enforces ExpectedNamespace/Capability/Function and runs
    PayloadValidator when provided. If everything is in order, Response (when set)
    provides the return bytes; otherwise it returns nil.

Tips

  - Use table-driven tests for different routing and payload cases.
  - Keep the validator small and focused—decode, assert, return.
  - Prefer component mocks unless you truly need wire-level checks.
*/
package hostmock
