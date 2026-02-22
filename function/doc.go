/*
Package function provides a client for invoking function-to-function callbacks
through the Tarmac host runtime.

The package exposes a minimal raw-bytes API: callers supply a function name and
input payload, and receive the target function output bytes.
*/
package function
