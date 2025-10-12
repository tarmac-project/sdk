/*
Package logging offers a client for emitting log entries from Tarmac WebAssembly
functions to the host runtime.

The package exposes a small interface with convenience methods for common log
levels (Info, Warn, Error, Debug, Trace). A client instance handles the host
interaction behind the scenes, so guest code can focus on writing logs.
*/
package logging
