package metrics

import ()

type Metrics interface {}

type metricsClient struct {}

type Config struct {}

func New(config *Config) (Metrics, error) {
  return &metricsClient{}, nil
}

type Counter struct {}

func (m *metricsClient) NewCounter(name string) (*Counter, error) {
  return &Counter{}, nil
}

func (c Counter) Inc() {}

type Histogram struct {}

func (m *metricsClient) NewHistogram(name string) (*Histogram, error) {
  return &Histogram{}
}

func (h Histogram) Observe(value float64) {}

type Gauge struct {}

func (m *metricsClient) NewGauge(name string) (*Gauge, error) {
  return &Gauge{}, nil
}

func (g *Gauge) Inc() {}
func (g *Gauge) Dec() {}
