package neo4j_tracing

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	TraceProvider trace.TracerProvider
	MeterProvider metric.MeterProvider
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		c.TraceProvider = tp
	})
}

// WithMeterProvider specifies a meter provider to use for recording metrics.
// If none is specified, metrics are not recorded.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		c.MeterProvider = mp
	})
}
