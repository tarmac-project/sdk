package metrics

import (
	"errors"
	"regexp"

	proto "github.com/tarmac-project/protobuf-go/sdk/metrics"
	sdk "github.com/tarmac-project/sdk"
	wapc "github.com/wapc/wapc-guest-tinygo"
)

const (
	capabilityName = "metrics"
	fnCounter      = "counter"
	fnGauge        = "gauge"
	fnHistogram    = "histogram"
	actionInc      = "inc"
	actionDec      = "dec"
)

var (
	// ErrInvalidMetricName indicates a metric name that does not match the supported format.
	ErrInvalidMetricName = errors.New("metric name is invalid")

	// isMetricNameValid validates metric names using the same pattern as tarmac callback validation.
	isMetricNameValid = regexp.MustCompile(`^[a-zA-Z0-9_:][a-zA-Z0-9_:]*$`)
)

// HostCall defines the waPC host function signature used by metrics operations.
type HostCall func(string, string, string, []byte) ([]byte, error)

// Client defines the metrics capability interface.
type Client interface {
	// NewCounter creates a named counter metric handle.
	NewCounter(name string) (*Counter, error)

	// NewGauge creates a named gauge metric handle.
	NewGauge(name string) (*Gauge, error)

	// NewHistogram creates a named histogram metric handle.
	NewHistogram(name string) (*Histogram, error)
}

// Config controls how a Client instance interacts with the host runtime.
type Config struct {
	// SDKConfig provides the runtime namespace used for host calls.
	SDKConfig sdk.RuntimeConfig

	// HostCall overrides the waPC host function used for metrics operations.
	HostCall HostCall
}

// HostMetrics is the metrics capability client implementation.
type HostMetrics struct {
	runtime  sdk.RuntimeConfig
	hostCall HostCall
}

// Counter is a named counter metric handle.
type Counter struct {
	name      string
	namespace string
	hostCall  HostCall
}

// Gauge is a named gauge metric handle.
type Gauge struct {
	name      string
	namespace string
	hostCall  HostCall
}

// Histogram is a named histogram metric handle.
type Histogram struct {
	name      string
	namespace string
	hostCall  HostCall
}

// Ensure HostMetrics satisfies the Client interface at compile time.
var _ Client = (*HostMetrics)(nil)

// New creates a metrics client with namespace defaults and optional host-call override.
func New(config Config) (*HostMetrics, error) {
	runtime := config.SDKConfig
	if runtime.Namespace == "" {
		runtime.Namespace = sdk.DefaultNamespace
	}

	hostCall := config.HostCall
	if hostCall == nil {
		hostCall = wapc.HostCall
	}

	return &HostMetrics{runtime: runtime, hostCall: hostCall}, nil
}

// NewCounter creates a named counter metric handle.
func (c *HostMetrics) NewCounter(name string) (*Counter, error) {
	if !isMetricNameValid.MatchString(name) {
		return nil, ErrInvalidMetricName
	}

	return &Counter{name: name, namespace: c.runtime.Namespace, hostCall: c.hostCall}, nil
}

// Inc increments the counter by one.
func (c *Counter) Inc() {
	payload, err := (&proto.MetricsCounter{Name: c.name}).MarshalVT()
	if err != nil {
		return
	}
	_, _ = c.hostCall(c.namespace, capabilityName, fnCounter, payload)
}

// NewGauge creates a named gauge metric handle.
func (c *HostMetrics) NewGauge(name string) (*Gauge, error) {
	if !isMetricNameValid.MatchString(name) {
		return nil, ErrInvalidMetricName
	}

	return &Gauge{name: name, namespace: c.runtime.Namespace, hostCall: c.hostCall}, nil
}

// Inc increments the gauge by one.
func (g *Gauge) Inc() {
	g.emit(actionInc)
}

// Dec decrements the gauge by one.
func (g *Gauge) Dec() {
	g.emit(actionDec)
}

// emit sends a gauge action update to the host runtime as a best-effort call.
func (g *Gauge) emit(action string) {
	payload, err := (&proto.MetricsGauge{Name: g.name, Action: action}).MarshalVT()
	if err != nil {
		return
	}
	_, _ = g.hostCall(g.namespace, capabilityName, fnGauge, payload)
}

// NewHistogram creates a named histogram metric handle.
func (c *HostMetrics) NewHistogram(name string) (*Histogram, error) {
	if !isMetricNameValid.MatchString(name) {
		return nil, ErrInvalidMetricName
	}

	return &Histogram{name: name, namespace: c.runtime.Namespace, hostCall: c.hostCall}, nil
}

// Observe records a value for the histogram.
func (h *Histogram) Observe(value float64) {
	payload, err := (&proto.MetricsHistogram{Name: h.name, Value: value}).MarshalVT()
	if err != nil {
		return
	}
	_, _ = h.hostCall(h.namespace, capabilityName, fnHistogram, payload)
}
