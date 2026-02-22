/*
Package sql provides a client for executing SQL operations through the Tarmac
host runtime.

The client supports Exec for statements that do not return rows and Query for
statements that return rows. Requests and responses are encoded with project
protobufs and sent through waPC host calls.

Errors are returned as package sentinels and SDK host errors so callers can use
errors.Is and errors.As for precise handling. Host partial-result responses are
surfaced as ErrPartialResult with a PartialResultError that retains operation
context and cause details.
*/
package sql
