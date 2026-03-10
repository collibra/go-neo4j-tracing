package neo4j_tracing

import (
	"context"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	accessModeAttrKey     = "db.neo4j.access_mode"
	bookmarksAttrKey      = "db.neo4j.bookmarks"
	bookmarksStartAttrKey = bookmarksAttrKey + ".start"
	bookmarksEndAttrKey   = bookmarksAttrKey + ".end"
	fetchAttrKey          = "db.neo4j.fetch"
)

type SessionAttributes struct {
	AccessMode   neo4j.AccessMode
	Bookmarks    neo4j.Bookmarks
	DatabaseName string
	FetchSize    int
}

// SessionTracer wraps a neo4j.SessionWithContext object so the calls can be traced with open telemetry distributed tracing
type SessionTracer struct {
	neo4j.Session

	tracer        trace.Tracer
	metrics       *metrics
	attributes    SessionAttributes
	serverAddress string
}

// BeginTransaction calls neo4j.SessionWithContext.BeginTransaction and trace the call
func (s *SessionTracer) BeginTransaction(ctx context.Context, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	start := time.Now()
	spanCtx, span := s.tracer.Start(ctx, "Session.BeginTransaction", trace.WithSpanKind(trace.SpanKindClient))

	s.attributes.SetAttributes(span)

	defer func() {
		span.SetAttributes(attribute.StringSlice(bookmarksEndAttrKey, s.LastBookmarks()))
	}()

	tx, err := s.Session.BeginTransaction(spanCtx, configurers...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		s.metrics.recordOperation(ctx, start, "BeginTransaction", s.attributes.DatabaseName, s.serverAddress, err)

		return nil, err
	}

	s.metrics.recordOperation(ctx, start, "BeginTransaction", s.attributes.DatabaseName, s.serverAddress, nil)

	return NewExplicitTransactionTracer(spanCtx, tx, span, s.tracer,
		WithTransactionMetrics(s.metrics),
		WithTransactionServerAddress(s.serverAddress),
		WithTransactionDBNamespace(s.attributes.DatabaseName),
	), nil
}

// ExecuteRead calls neo4j.SessionWithContext.ExecuteRead and trace the call.
// The neo4j.ManagedTransaction object that is passed to the work function will be wrapped with a tracer.
func (s *SessionTracer) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
	return s.execute(ctx, "ExecuteRead", s.Session.ExecuteRead, work, configurers...)
}

// ExecuteWrite calls neo4j.SessionWithContext.ExecuteWrite and trace the call.
// The neo4j.ManagedTransaction object that is passed to the work function will be wrapped with a tracer.
func (s *SessionTracer) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
	return s.execute(ctx, "ExecuteWrite", s.Session.ExecuteWrite, work, configurers...)
}

func (s *SessionTracer) execute(ctx context.Context,
	spanOperation string, f func(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error),
	work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig),
) (_ any, err error) {
	start := time.Now()
	spanCtx, span := s.tracer.Start(ctx, spanName(spanOperation), trace.WithSpanKind(trace.SpanKindClient))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
		s.metrics.recordOperation(ctx, start, spanOperation, s.attributes.DatabaseName, s.serverAddress, err)
	}()

	s.attributes.SetAttributes(span)

	defer func() {
		span.SetAttributes(attribute.StringSlice(bookmarksEndAttrKey, s.LastBookmarks()))
	}()

	return f(spanCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		txTracing := NewManagedTransactionTracer(spanCtx, tx, s.tracer,
			WithTransactionMetrics(s.metrics),
			WithTransactionServerAddress(s.serverAddress),
			WithTransactionDBNamespace(s.attributes.DatabaseName),
		)

		return work(txTracing)
	}, configurers...)
}

// Run calls neo4j.SessionWithContext.Run and trace the call
func (s *SessionTracer) Run(ctx context.Context, cypher string, params map[string]any, configurers ...func(config *neo4j.TransactionConfig)) (_ neo4j.Result, err error) {
	start := time.Now()
	spanCtx, span := s.tracer.Start(ctx, spanName("Run"), trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(semconv.DBStatement(cypher), semconv.DBSystemNeo4j))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
		s.metrics.recordOperation(ctx, start, "Run", s.attributes.DatabaseName, s.serverAddress, err)
	}()

	s.attributes.SetAttributes(span)

	defer func() {
		span.SetAttributes(attribute.StringSlice(bookmarksEndAttrKey, s.LastBookmarks()))
	}()

	result, err := s.Session.Run(spanCtx, cypher, params, configurers...)

	return NewResultTracer(spanCtx, result, s.tracer,
		WithResultMetrics(s.metrics),
		WithResultServerAddress(s.serverAddress),
		WithResultDBNamespace(s.attributes.DatabaseName),
	), err
}

func (a SessionAttributes) SetAttributes(span trace.Span) {
	accessMode := "READ"
	if a.AccessMode == neo4j.AccessModeWrite {
		accessMode = "WRITE"
	}

	span.SetAttributes(
		attribute.StringSlice(bookmarksStartAttrKey, a.Bookmarks),
		semconv.DBName(a.DatabaseName),
		attribute.Int(fetchAttrKey, a.FetchSize),
		attribute.String(accessModeAttrKey, accessMode),
	)
}
