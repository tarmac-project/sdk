/*
Package kv provides a client for interacting with the Tarmac key-value
capability from WebAssembly guest functions.

The client serializes requests with project protobufs, forwards them to the
host with waPC, and returns structured responses. Zero-value Config options
fall back to sensible defaults such as the `sdk.DefaultNamespace` and the
default waPC host call.

Typical usage is to construct a Client with New, then invoke Set, Get, Delete,
and Keys. Tests can inject custom host behaviour with Config.HostCall to
exercise failure paths without a real host.
*/
package kv
