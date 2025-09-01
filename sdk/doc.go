/*
Package sdk provides the core entry point and runtime configuration for
building Tarmac WebAssembly functions.

The package exposes New to register a waPC handler and a RuntimeConfig that is
shared by capability clients (e.g., HTTP, KV). DefaultNamespace is used when a
namespace is not explicitly provided.
*/
package sdk

