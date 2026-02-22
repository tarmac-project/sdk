/*
Package metrics provides a client for creating custom metrics through the
Tarmac host runtime.

The package exposes constructors for Counter, Gauge, and Histogram metric
handles, each backed by protobuf payloads sent over waPC host calls.

Metric emission methods intentionally follow Prometheus-style ergonomics:
Inc/Dec/Observe are best-effort and do not return errors. Marshal or host-call
failures are treated as non-fatal and are swallowed to avoid impacting caller
control flow.
*/
package metrics
