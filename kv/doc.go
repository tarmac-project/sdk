/*
Package kv provides a key-value client for Tarmac functions.

It communicates with the host via waPC using protobuf payloads and implements
Get, Set, Delete, and Keys operations. Errors include invalid inputs,
host-call failures, and not-found conditions.
*/
package kv

