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
	propertiesSet        metric.Int64Counter
	labelsAdded          metric.Int64Counter
	labelsRemoved        metric.Int64Counter
	indexesAdded         metric.Int64Counter
	indexesRemoved       metric.Int64Counter
	constraintsAdded     metric.Int64Counter
	constraintsRemoved   metric.Int64Counter
	systemUpdates        metric.Int64Counter
	// Session lifecycle
	sessionCount  metric.Int64Counter
	sessionActive metric.Int64UpDownCounter
	// Transaction lifecycle
	transactionCount    metric.Int64Counter
	transactionCommit   metric.Int64Counter
	transactionRollback metric.Int64Counter
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

	propertiesSet, _ := meter.Int64Counter("db.client.result.properties_set",
		metric.WithDescription("Cumulative properties set"),
	)

	labelsAdded, _ := meter.Int64Counter("db.client.result.labels_added",
		metric.WithDescription("Cumulative labels added"),
	)

	labelsRemoved, _ := meter.Int64Counter("db.client.result.labels_removed",
		metric.WithDescription("Cumulative labels removed"),
	)

	indexesAdded, _ := meter.Int64Counter("db.client.result.indexes_added",
		metric.WithDescription("Cumulative indexes added"),
	)

	indexesRemoved, _ := meter.Int64Counter("db.client.result.indexes_removed",
		metric.WithDescription("Cumulative indexes removed"),
	)

	constraintsAdded, _ := meter.Int64Counter("db.client.result.constraints_added",
		metric.WithDescription("Cumulative constraints added"),
	)

	constraintsRemoved, _ := meter.Int64Counter("db.client.result.constraints_removed",
		metric.WithDescription("Cumulative constraints removed"),
	)

	systemUpdates, _ := meter.Int64Counter("db.client.result.system_updates",
		metric.WithDescription("Cumulative system updates"),
	)

	sessionCount, _ := meter.Int64Counter("db.client.session.count",
		metric.WithDescription("Total sessions created"),
	)

	sessionActive, _ := meter.Int64UpDownCounter("db.client.session.active",
		metric.WithDescription("Currently active sessions"),
	)

	transactionCount, _ := meter.Int64Counter("db.client.transaction.count",
		metric.WithDescription("Total transactions started"),
	)

	transactionCommit, _ := meter.Int64Counter("db.client.transaction.commit.count",
		metric.WithDescription("Committed transactions"),
	)

	transactionRollback, _ := meter.Int64Counter("db.client.transaction.rollback.count",
		metric.WithDescription("Rolled back transactions"),
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
		propertiesSet:        propertiesSet,
		labelsAdded:          labelsAdded,
		labelsRemoved:        labelsRemoved,
		indexesAdded:         indexesAdded,
		indexesRemoved:       indexesRemoved,
		constraintsAdded:     constraintsAdded,
		constraintsRemoved:   constraintsRemoved,
		systemUpdates:        systemUpdates,
		sessionCount:         sessionCount,
		sessionActive:        sessionActive,
		transactionCount:     transactionCount,
		transactionCommit:    transactionCommit,
		transactionRollback:  transactionRollback,
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
		m.propertiesSet.Add(ctx, int64(counters.PropertiesSet()), opt)
		m.labelsAdded.Add(ctx, int64(counters.LabelsAdded()), opt)
		m.labelsRemoved.Add(ctx, int64(counters.LabelsRemoved()), opt)
		m.indexesAdded.Add(ctx, int64(counters.IndexesAdded()), opt)
		m.indexesRemoved.Add(ctx, int64(counters.IndexesRemoved()), opt)
		m.constraintsAdded.Add(ctx, int64(counters.ConstraintsAdded()), opt)
		m.constraintsRemoved.Add(ctx, int64(counters.ConstraintsRemoved()), opt)
		m.systemUpdates.Add(ctx, int64(counters.SystemUpdates()), opt)
	}
}

func (m *metrics) recordSessionOpen(ctx context.Context, serverAddress string) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrDBSystemName, serviceID),
	}

	if serverAddress != "" {
		attrs = append(attrs, attribute.String(attrServerAddress, serverAddress))
	}

	opt := metric.WithAttributes(attrs...)

	m.sessionCount.Add(ctx, 1, opt)
	m.sessionActive.Add(ctx, 1, opt)
}

func (m *metrics) recordSessionClose(ctx context.Context, serverAddress string) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String(attrDBSystemName, serviceID),
	}

	if serverAddress != "" {
		attrs = append(attrs, attribute.String(attrServerAddress, serverAddress))
	}

	opt := metric.WithAttributes(attrs...)

	m.sessionActive.Add(ctx, -1, opt)
}

func (m *metrics) recordTransactionStart(ctx context.Context, dbNamespace, serverAddress string) {
	if m == nil {
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

	m.transactionCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (m *metrics) recordTransactionCommit(ctx context.Context, dbNamespace, serverAddress string) {
	if m == nil {
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

	m.transactionCommit.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (m *metrics) recordTransactionRollback(ctx context.Context, dbNamespace, serverAddress string) {
	if m == nil {
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

	m.transactionRollback.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func errorType(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%T", err)
}
