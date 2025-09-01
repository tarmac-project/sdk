/*
Package http provides an HTTP client for Tarmac WebAssembly functions.

Requests are serialized via protobuf and sent to the host using waPC. The
Client interface offers convenience methods (Get, Post, Put, Delete) and a Do
method for custom requests. Errors use sentinel values combined with the
underlying cause and can be checked with errors.Is.
*/
package http

