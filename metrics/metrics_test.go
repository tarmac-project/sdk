package metrics

import (
	"errors"
	"reflect"
	"testing"

	proto "github.com/tarmac-project/protobuf-go/sdk/metrics"
	sdk "github.com/tarmac-project/sdk"
	"github.com/tarmac-project/sdk/hostmock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	customHostCall := func(string, string, string, []byte) ([]byte, error) {
		return nil, nil
	}

	tt := []struct {
		name        string
		namespace   string
		hostCall    HostCall
		wantNS      string
		wantHostPtr uintptr
	}{
		{
			name:      "custom namespace",
			namespace: "custom",
			wantNS:    "custom",
		},
		{
			name:        "default namespace with override",
			hostCall:    customHostCall,
			wantNS:      sdk.DefaultNamespace,
			wantHostPtr: reflect.ValueOf(customHostCall).Pointer(),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: tc.namespace}, HostCall: tc.hostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			if c.runtime.Namespace != tc.wantNS {
				t.Fatalf("namespace mismatch: want %q, got %q", tc.wantNS, c.runtime.Namespace)
			}

			if tc.wantHostPtr != 0 {
				if got := reflect.ValueOf(c.hostCall).Pointer(); got != tc.wantHostPtr {
					t.Fatalf("hostcall pointer mismatch: want %v, got %v", tc.wantHostPtr, got)
				}
			}
		})
	}
}

func TestMetricConstructors(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"},
		HostCall: func(string, string, string, []byte) ([]byte, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	tt := []struct {
		name        string
		constructor func(string) error
		metricName  string
		wantErr     error
	}{
		{
			name: "counter valid",
			constructor: func(name string) error {
				_, callErr := c.NewCounter(name)
				return callErr
			},
			metricName: "requests_total",
		},
		{
			name: "gauge valid",
			constructor: func(name string) error {
				_, callErr := c.NewGauge(name)
				return callErr
			},
			metricName: "queue_depth",
		},
		{
			name: "histogram valid",
			constructor: func(name string) error {
				_, callErr := c.NewHistogram(name)
				return callErr
			},
			metricName: "request_duration",
		},
		{
			name: "counter empty name",
			constructor: func(name string) error {
				_, callErr := c.NewCounter(name)
				return callErr
			},
			metricName: "",
			wantErr:    ErrInvalidMetricName,
		},
		{
			name: "gauge whitespace name",
			constructor: func(name string) error {
				_, callErr := c.NewGauge(name)
				return callErr
			},
			metricName: " \n\t ",
			wantErr:    ErrInvalidMetricName,
		},
		{
			name: "histogram empty name",
			constructor: func(name string) error {
				_, callErr := c.NewHistogram(name)
				return callErr
			},
			metricName: "",
			wantErr:    ErrInvalidMetricName,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotErr := tc.constructor(tc.metricName)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("unexpected error: want %v got %v", tc.wantErr, gotErr)
			}
		})
	}
}

func TestCounterInc(t *testing.T) {
	t.Parallel()

	cfg := hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: capabilityName,
		ExpectedFunction:   fnCounter,
		PayloadValidator: func(payload []byte) error {
			var req proto.MetricsCounter
			if err := req.UnmarshalVT(payload); err != nil {
				return err
			}
			if req.GetName() != "requests_total" {
				return errors.New("metric name mismatch")
			}
			return nil
		},
		Fail:  true,
		Error: errors.New("host failure should not panic"),
	}

	mock, err := hostmock.New(cfg)
	if err != nil {
		t.Fatalf("failed to create hostmock: %v", err)
	}

	c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"}, HostCall: mock.HostCall})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	counter, err := c.NewCounter("requests_total")
	if err != nil {
		t.Fatalf("NewCounter returned error: %v", err)
	}

	counter.Inc()
}

func TestGaugeActions(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name           string
		invoke         func(*Gauge)
		expectedAction string
	}{
		{
			name: "inc",
			invoke: func(g *Gauge) {
				g.Inc()
			},
			expectedAction: actionInc,
		},
		{
			name: "dec",
			invoke: func(g *Gauge) {
				g.Dec()
			},
			expectedAction: actionDec,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := hostmock.Config{
				ExpectedNamespace:  "tarmac",
				ExpectedCapability: capabilityName,
				ExpectedFunction:   fnGauge,
				PayloadValidator: func(payload []byte) error {
					var req proto.MetricsGauge
					if err := req.UnmarshalVT(payload); err != nil {
						return err
					}
					if req.GetName() != "queue_depth" {
						return errors.New("metric name mismatch")
					}
					if req.GetAction() != tc.expectedAction {
						return errors.New("action mismatch")
					}
					return nil
				},
			}

			mock, err := hostmock.New(cfg)
			if err != nil {
				t.Fatalf("failed to create hostmock: %v", err)
			}

			c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"}, HostCall: mock.HostCall})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			gauge, err := c.NewGauge("queue_depth")
			if err != nil {
				t.Fatalf("NewGauge returned error: %v", err)
			}

			tc.invoke(gauge)
		})
	}
}

func TestHistogramObserve(t *testing.T) {
	t.Parallel()

	cfg := hostmock.Config{
		ExpectedNamespace:  "tarmac",
		ExpectedCapability: capabilityName,
		ExpectedFunction:   fnHistogram,
		PayloadValidator: func(payload []byte) error {
			var req proto.MetricsHistogram
			if err := req.UnmarshalVT(payload); err != nil {
				return err
			}
			if req.GetName() != "request_duration" {
				return errors.New("metric name mismatch")
			}
			if req.GetValue() != 42.5 {
				return errors.New("metric value mismatch")
			}
			return nil
		},
		Fail:  true,
		Error: errors.New("host failure should not panic"),
	}

	mock, err := hostmock.New(cfg)
	if err != nil {
		t.Fatalf("failed to create hostmock: %v", err)
	}

	c, err := New(Config{SDKConfig: sdk.RuntimeConfig{Namespace: "tarmac"}, HostCall: mock.HostCall})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	histogram, err := c.NewHistogram("request_duration")
	if err != nil {
		t.Fatalf("NewHistogram returned error: %v", err)
	}

	histogram.Observe(42.5)
}
