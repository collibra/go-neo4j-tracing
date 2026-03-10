package neo4j_tracing

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type resultTracerConfig struct {
	metrics       *metrics
	serverAddress string
	dbNamespace   string
}

// ResultTracerOption configures optional fields for the result tracer.
type ResultTracerOption func(*resultTracerConfig)

// WithResultMetrics sets the metrics recorder for the result tracer.
func WithResultMetrics(m *metrics) ResultTracerOption {
	return func(c *resultTracerConfig) {
		c.metrics = m
	}
}

// WithResultServerAddress sets the server address attribute for metrics.
func WithResultServerAddress(addr string) ResultTracerOption {
	return func(c *resultTracerConfig) {
		c.serverAddress = addr
	}
}

// WithResultDBNamespace sets the database namespace attribute for metrics.
func WithResultDBNamespace(ns string) ResultTracerOption {
	return func(c *resultTracerConfig) {
		c.dbNamespace = ns
	}
}

type ResultTracer struct {
	neo4j.Result

	ctx           context.Context
	tracer        trace.Tracer
	metrics       *metrics
	serverAddress string
	dbNamespace   string
}

func NewResultTracer(ctx context.Context, result neo4j.Result, tracer trace.Tracer, opts ...ResultTracerOption) neo4j.Result {
	cfg := resultTracerConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	return &ResultTracer{
		Result:        result,
		ctx:           ctx,
		tracer:        tracer,
		metrics:       cfg.metrics,
		serverAddress: cfg.serverAddress,
		dbNamespace:   cfg.dbNamespace,
	}
}

func (r *ResultTracer) NextRecord(ctx context.Context, record **neo4j.Record) bool {
	_, span := r.tracer.Start(r.ctx, spanName("Record.NextRecord"), trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	return r.Result.NextRecord(ctx, record)
}

func (r *ResultTracer) Next(ctx context.Context) bool {
	_, span := r.tracer.Start(r.ctx, spanName("Record.Next"), trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	return r.Result.Next(ctx)
}

func (r *ResultTracer) PeekRecord(ctx context.Context, record **neo4j.Record) bool {
	_, span := r.tracer.Start(r.ctx, spanName("Record.PeekRecord"), trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	return r.Result.PeekRecord(ctx, record)
}

func (r *ResultTracer) Peek(ctx context.Context) bool {
	_, span := r.tracer.Start(r.ctx, spanName("Record.Peek"), trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	return r.Result.Peek(ctx)
}

func (r *ResultTracer) Collect(ctx context.Context) (_ []*neo4j.Record, err error) {
	_, span := r.tracer.Start(r.ctx, spanName("Record.Collect"), trace.WithSpanKind(trace.SpanKindInternal))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}()

	return r.Result.Collect(ctx)
}

func (r *ResultTracer) Single(ctx context.Context) (_ *neo4j.Record, err error) {
	_, span := r.tracer.Start(r.ctx, spanName("Record.Single"), trace.WithSpanKind(trace.SpanKindInternal))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}()

	record, err := r.Result.Single(ctx)
	if err != nil {
		return record, err
	}

	summary, consumeErr := r.Result.Consume(ctx)
	if consumeErr == nil {
		r.metrics.recordResultSummary(ctx, summary, r.dbNamespace, r.serverAddress)
	}

	return record, err
}

func (r *ResultTracer) Consume(ctx context.Context) (_ neo4j.ResultSummary, err error) {
	_, span := r.tracer.Start(r.ctx, spanName("Record.Consume"), trace.WithSpanKind(trace.SpanKindInternal))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}()

	summary, err := r.Result.Consume(ctx)
	if err == nil {
		r.metrics.recordResultSummary(ctx, summary, r.dbNamespace, r.serverAddress)
	}

	return summary, err
}
