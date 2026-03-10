package neo4j_tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterName = "github.com/collibra/go-neo4j-tracing"

	attrDBSystemName  = "db.system.name"
	attrDBOpName      = "db.operation.name"
	attrDBNamespace   = "db.namespace"
	attrServerAddress = "server.address"
	attrErrorType     = "error.type"
)

type metrics struct {
	// Core
	operationDuration metric.Float64Histogram
	operationCount    metric.Int64Counter
	errorCount        metric.Int64Counter
	// ResultSummary
	resultAvailableAfter metric.Float64Histogram
	resultConsumedAfter  metric.Float64Histogram
	nodesCreated         metric.Int64Counter
	nodesDeleted         metric.Int64Counter
	relationshipsCreated metric.Int64Counter
	relationshipsDeleted metric.Int64Counter
}

var histogramBuckets = metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10)

func newMetrics(mp metric.MeterProvider) *metrics {
	if mp == nil {
		return nil
	}

	meter := mp.Meter(meterName)

	operationDuration, _ := meter.Float64Histogram("db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of client-side database operations"),
		histogramBuckets,
	)

	operationCount, _ := meter.Int64Counter("db.client.operation.count",
		metric.WithDescription("Total number of database operations executed"),
	)

	errorCount, _ := meter.Int64Counter("db.client.error.count",
		metric.WithDescription("Total number of failed database operations"),
	)

	resultAvailableAfter, _ := meter.Float64Histogram("db.client.result.available_after",
		metric.WithUnit("s"),
		metric.WithDescription("Server-side time until result was available"),
		histogramBuckets,
	)

	resultConsumedAfter, _ := meter.Float64Histogram("db.client.result.consumed_after",
		metric.WithUnit("s"),
		metric.WithDescription("Server-side time to consume result"),
		histogramBuckets,
	)

	nodesCreated, _ := meter.Int64Counter("db.client.result.nodes_created",
		metric.WithDescription("Cumulative nodes created"),
	)

	nodesDeleted, _ := meter.Int64Counter("db.client.result.nodes_deleted",
		metric.WithDescription("Cumulative nodes deleted"),
	)

	relationshipsCreated, _ := meter.Int64Counter("db.client.result.relationships_created",
		metric.WithDescription("Cumulative relationships created"),
	)

	relationshipsDeleted, _ := meter.Int64Counter("db.client.result.relationships_deleted",
		metric.WithDescription("Cumulative relationships deleted"),
	)

	return &metrics{
		operationDuration:    operationDuration,
		operationCount:       operationCount,
		errorCount:           errorCount,
		resultAvailableAfter: resultAvailableAfter,
		resultConsumedAfter:  resultConsumedAfter,
		nodesCreated:         nodesCreated,
		nodesDeleted:         nodesDeleted,
		relationshipsCreated: relationshipsCreated,
		relationshipsDeleted: relationshipsDeleted,
	}
}

func (m *metrics) recordOperation(ctx context.Context, start time.Time, operation, dbNamespace, serverAddress string, err error) {
	if m == nil {
		return
	}

	duration := time.Since(start).Seconds()

	attrs := []attribute.KeyValue{
		attribute.String(attrDBSystemName, serviceID),
		attribute.String(attrDBOpName, operation),
	}

	if dbNamespace != "" {
		attrs = append(attrs, attribute.String(attrDBNamespace, dbNamespace))
	}

	if serverAddress != "" {
		attrs = append(attrs, attribute.String(attrServerAddress, serverAddress))
	}

	if err != nil {
		attrs = append(attrs, attribute.String(attrErrorType, errorType(err)))
	}

	opt := metric.WithAttributes(attrs...)

	m.operationDuration.Record(ctx, duration, opt)
	m.operationCount.Add(ctx, 1, opt)

	if err != nil {
		m.errorCount.Add(ctx, 1, opt)
	}
}

func (m *metrics) recordResultSummary(ctx context.Context, summary neo4j.ResultSummary, dbNamespace, serverAddress string) {
	if m == nil {
		return
	}

	if summary == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrDBSystemName, serviceID),
	}

	if dbNamespace != "" {
		attrs = append(attrs, attribute.String(attrDBNamespace, dbNamespace))
	}

	if serverAddress != "" {
		attrs = append(attrs, attribute.String(attrServerAddress, serverAddress))
	}

	opt := metric.WithAttributes(attrs...)

	m.resultAvailableAfter.Record(ctx, summary.ResultAvailableAfter().Seconds(), opt)
	m.resultConsumedAfter.Record(ctx, summary.ResultConsumedAfter().Seconds(), opt)

	counters := summary.Counters()
	if counters != nil {
		m.nodesCreated.Add(ctx, int64(counters.NodesCreated()), opt)
		m.nodesDeleted.Add(ctx, int64(counters.NodesDeleted()), opt)
		m.relationshipsCreated.Add(ctx, int64(counters.RelationshipsCreated()), opt)
		m.relationshipsDeleted.Add(ctx, int64(counters.RelationshipsDeleted()), opt)
	}
}

func errorType(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%T", err)
}
