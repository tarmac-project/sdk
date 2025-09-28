/*
Package kv provides a client for interacting with the Tarmac key-value
capability from WebAssembly guest functions.

The package focuses on creating clients that can be wired into tests via
custom host call functions while defaulting to the Tarmac namespace when none
is provided.
*/
package kv
