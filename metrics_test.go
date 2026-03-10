package neo4j_tracing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewMetrics_NilProvider(t *testing.T) {
	m := newMetrics(nil)
	assert.Nil(t, m)
}

func TestNewMetrics_WithProvider(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	m := newMetrics(mp)
	assert.NotNil(t, m)
	assert.NotNil(t, m.operationDuration)
	assert.NotNil(t, m.operationCount)
	assert.NotNil(t, m.errorCount)
	assert.NotNil(t, m.resultAvailableAfter)
	assert.NotNil(t, m.resultConsumedAfter)
	assert.NotNil(t, m.nodesCreated)
	assert.NotNil(t, m.nodesDeleted)
	assert.NotNil(t, m.relationshipsCreated)
	assert.NotNil(t, m.relationshipsDeleted)
	assert.NotNil(t, m.propertiesSet)
	assert.NotNil(t, m.labelsAdded)
	assert.NotNil(t, m.labelsRemoved)
	assert.NotNil(t, m.indexesAdded)
	assert.NotNil(t, m.indexesRemoved)
	assert.NotNil(t, m.constraintsAdded)
	assert.NotNil(t, m.constraintsRemoved)
	assert.NotNil(t, m.systemUpdates)
	assert.NotNil(t, m.sessionCount)
	assert.NotNil(t, m.sessionActive)
	assert.NotNil(t, m.transactionCount)
	assert.NotNil(t, m.transactionCommit)
	assert.NotNil(t, m.transactionRollback)
}

func TestMetrics_RecordOperation_NilSafe(t *testing.T) {
	var m *metrics
	// Should not panic
	m.recordOperation(context.Background(), time.Now(), "Run", "testdb", "localhost", nil)
}

func TestMetrics_RecordResultSummary_NilSafe(t *testing.T) {
	var m *metrics
	// Should not panic
	m.recordResultSummary(context.Background(), nil, "testdb", "localhost")
}

func TestMetrics_RecordOperation_Success(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()
	m.recordOperation(ctx, time.Now().Add(-10*time.Millisecond), "Run", "testdb", "localhost", nil)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	// Check operation count
	opCount, ok := metrics["db.client.operation.count"]
	require.True(t, ok, "db.client.operation.count metric should exist")
	assertCounterValue(t, opCount, 1)
	assertHasAttribute(t, opCount, attrDBSystemName, "neo4j")
	assertHasAttribute(t, opCount, attrDBOpName, "Run")
	assertHasAttribute(t, opCount, attrDBNamespace, "testdb")
	assertHasAttribute(t, opCount, attrServerAddress, "localhost")

	// Check duration exists
	_, ok = metrics["db.client.operation.duration"]
	assert.True(t, ok, "db.client.operation.duration metric should exist")

	// Error count should not have data points for successful operations
	errCount, ok := metrics["db.client.error.count"]
	if ok {
		assertCounterValue(t, errCount, 0)
	}
}

func TestMetrics_RecordOperation_Error(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()
	testErr := errors.New("test error")
	m.recordOperation(ctx, time.Now().Add(-10*time.Millisecond), "Run", "testdb", "localhost", testErr)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	// Check error count
	errCount, ok := metrics["db.client.error.count"]
	require.True(t, ok, "db.client.error.count metric should exist")
	assertCounterValue(t, errCount, 1)
	assertHasAttribute(t, errCount, attrErrorType, "*errors.errorString")

	// Check operation count also incremented
	opCount, ok := metrics["db.client.operation.count"]
	require.True(t, ok)
	assertCounterValue(t, opCount, 1)
}

func TestMetrics_RecordOperation_NoNamespace(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()
	m.recordOperation(ctx, time.Now(), "VerifyConnectivity", "", "localhost", nil)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)
	opCount := metrics["db.client.operation.count"]
	assertDoesNotHaveAttribute(t, opCount, attrDBNamespace)
}

func TestMetrics_RecordResultSummary(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()
	summary := &mockResultSummaryWithCounters{
		availableAfter: 50 * time.Millisecond,
		consumedAfter:  100 * time.Millisecond,
		counters: &mockCounters{
			nodesCreated:         3,
			nodesDeleted:         1,
			relationshipsCreated: 2,
			relationshipsDeleted: 0,
		},
	}

	m.recordResultSummary(ctx, summary, "testdb", "localhost")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	_, ok := metrics["db.client.result.available_after"]
	assert.True(t, ok, "db.client.result.available_after metric should exist")

	_, ok = metrics["db.client.result.consumed_after"]
	assert.True(t, ok, "db.client.result.consumed_after metric should exist")

	nodesCreated, ok := metrics["db.client.result.nodes_created"]
	require.True(t, ok)
	assertCounterValue(t, nodesCreated, 3)

	nodesDeleted, ok := metrics["db.client.result.nodes_deleted"]
	require.True(t, ok)
	assertCounterValue(t, nodesDeleted, 1)

	relsCreated, ok := metrics["db.client.result.relationships_created"]
	require.True(t, ok)
	assertCounterValue(t, relsCreated, 2)

	relsDeleted, ok := metrics["db.client.result.relationships_deleted"]
	require.True(t, ok)
	assertCounterValue(t, relsDeleted, 0)
}

func TestMetrics_RecordResultSummary_NilSummary(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	// Should not panic
	m.recordResultSummary(context.Background(), nil, "testdb", "localhost")
}

func TestMetrics_RecordResultSummary_NilCounters(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	summary := &mockResultSummaryWithCounters{
		availableAfter: 50 * time.Millisecond,
		consumedAfter:  100 * time.Millisecond,
		counters:       nil,
	}

	// Should not panic
	m.recordResultSummary(context.Background(), summary, "testdb", "localhost")
}

func TestErrorType(t *testing.T) {
	assert.Empty(t, errorType(nil))
	assert.Equal(t, "*errors.errorString", errorType(errors.New("test")))
}

func TestWithMeterProvider(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	tracer := NewNeo4jTracer(WithMeterProvider(mp))
	assert.NotNil(t, tracer.metrics)
}

func TestWithoutMeterProvider_NoMetrics(t *testing.T) {
	tracer := NewNeo4jTracer()
	assert.Nil(t, tracer.metrics)
}

func TestParseServerAddress(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"neo4j://localhost", "localhost"},
		{"neo4j://localhost:7687", "localhost"},
		{"bolt://myhost.example.com:7687", "myhost.example.com"},
		{"neo4j+s://aura.example.com", "aura.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseServerAddress(tt.target))
		})
	}
}

func TestMetrics_ConsumeRecordsResultSummary(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	m := newMetrics(mp)
	ctx := context.Background()

	summary := &mockResultSummaryWithCounters{
		availableAfter: 25 * time.Millisecond,
		consumedAfter:  50 * time.Millisecond,
		counters: &mockCounters{
			nodesCreated: 5,
		},
	}

	resultTracer := &ResultTracer{
		Result: &mockResult{
			consumeFunc: func(ctx context.Context) (neo4j.ResultSummary, error) {
				return summary, nil
			},
		},
		ctx:           ctx,
		tracer:        noopTracer(),
		metrics:       m,
		serverAddress: "localhost",
		dbNamespace:   "testdb",
	}

	_, err := resultTracer.Consume(ctx)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	nodesCreated, ok := metrics["db.client.result.nodes_created"]
	require.True(t, ok)
	assertCounterValue(t, nodesCreated, 5)
}

func TestMetrics_ConsumeError_NoResultSummary(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	m := newMetrics(mp)
	ctx := context.Background()

	resultTracer := &ResultTracer{
		Result: &mockResult{
			consumeFunc: func(ctx context.Context) (neo4j.ResultSummary, error) {
				return nil, errors.New("consume failed")
			},
		},
		ctx:           ctx,
		tracer:        noopTracer(),
		metrics:       m,
		serverAddress: "localhost",
		dbNamespace:   "testdb",
	}

	_, err := resultTracer.Consume(ctx)
	require.Error(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	// No result summary metrics should be recorded on error
	_, ok := metrics["db.client.result.nodes_created"]
	assert.False(t, ok)
}

func TestMetrics_DriverVerifyConnectivity_WithMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	m := newMetrics(mp)
	ctx := context.Background()

	driverTracer := &DriverTracer{
		Driver: &mockDriver{
			verifyConnectivityFunc: func(ctx context.Context) error {
				return nil
			},
		},
		tracer:        noopTracer(),
		metrics:       m,
		serverAddress: "localhost",
	}

	err := driverTracer.VerifyConnectivity(ctx)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	opCount, ok := metrics["db.client.operation.count"]
	require.True(t, ok)
	assertCounterValue(t, opCount, 1)
	assertHasAttribute(t, opCount, attrDBOpName, "VerifyConnectivity")
}

func TestMetrics_RecordResultSummary_AllCounters(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()
	summary := &mockResultSummaryWithCounters{
		availableAfter: 50 * time.Millisecond,
		consumedAfter:  100 * time.Millisecond,
		counters: &mockCounters{
			nodesCreated:         3,
			nodesDeleted:         1,
			relationshipsCreated: 2,
			relationshipsDeleted: 0,
			propertiesSet:        5,
			labelsAdded:          4,
			labelsRemoved:        2,
			indexesAdded:         1,
			indexesRemoved:       0,
			constraintsAdded:     1,
			constraintsRemoved:   0,
			systemUpdates:        3,
		},
	}

	m.recordResultSummary(ctx, summary, "testdb", "localhost")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	propertiesSet, ok := metrics["db.client.result.properties_set"]
	require.True(t, ok)
	assertCounterValue(t, propertiesSet, 5)

	labelsAdded, ok := metrics["db.client.result.labels_added"]
	require.True(t, ok)
	assertCounterValue(t, labelsAdded, 4)

	labelsRemoved, ok := metrics["db.client.result.labels_removed"]
	require.True(t, ok)
	assertCounterValue(t, labelsRemoved, 2)

	indexesAdded, ok := metrics["db.client.result.indexes_added"]
	require.True(t, ok)
	assertCounterValue(t, indexesAdded, 1)

	indexesRemoved, ok := metrics["db.client.result.indexes_removed"]
	require.True(t, ok)
	assertCounterValue(t, indexesRemoved, 0)

	constraintsAdded, ok := metrics["db.client.result.constraints_added"]
	require.True(t, ok)
	assertCounterValue(t, constraintsAdded, 1)

	constraintsRemoved, ok := metrics["db.client.result.constraints_removed"]
	require.True(t, ok)
	assertCounterValue(t, constraintsRemoved, 0)

	systemUpdates, ok := metrics["db.client.result.system_updates"]
	require.True(t, ok)
	assertCounterValue(t, systemUpdates, 3)
}

func TestMetrics_RecordSessionOpenClose(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()

	m.recordSessionOpen(ctx, "localhost")
	m.recordSessionOpen(ctx, "localhost")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	sessionCount, ok := metrics["db.client.session.count"]
	require.True(t, ok)
	assertCounterValue(t, sessionCount, 2)

	sessionActive, ok := metrics["db.client.session.active"]
	require.True(t, ok)
	assertUpDownCounterValue(t, sessionActive, 2)

	// Close one session
	m.recordSessionClose(ctx, "localhost")

	rm = metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics = collectMetricsByName(rm)

	sessionActive, ok = metrics["db.client.session.active"]
	require.True(t, ok)
	assertUpDownCounterValue(t, sessionActive, 1)
}

func TestMetrics_RecordSessionOpen_NilSafe(t *testing.T) {
	var m *metrics
	// Should not panic
	m.recordSessionOpen(context.Background(), "localhost")
}

func TestMetrics_RecordSessionClose_NilSafe(t *testing.T) {
	var m *metrics
	// Should not panic
	m.recordSessionClose(context.Background(), "localhost")
}

func TestMetrics_RecordTransactionLifecycle(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)

	ctx := context.Background()

	m.recordTransactionStart(ctx, "testdb", "localhost")
	m.recordTransactionStart(ctx, "testdb", "localhost")
	m.recordTransactionCommit(ctx, "testdb", "localhost")
	m.recordTransactionRollback(ctx, "testdb", "localhost")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	metrics := collectMetricsByName(rm)

	txCount, ok := metrics["db.client.transaction.count"]
	require.True(t, ok)
	assertCounterValue(t, txCount, 2)

	commitCount, ok := metrics["db.client.transaction.commit.count"]
	require.True(t, ok)
	assertCounterValue(t, commitCount, 1)

	rollbackCount, ok := metrics["db.client.transaction.rollback.count"]
	require.True(t, ok)
	assertCounterValue(t, rollbackCount, 1)
}

func TestMetrics_RecordTransaction_NilSafe(t *testing.T) {
	var m *metrics
	// Should not panic
	m.recordTransactionStart(context.Background(), "testdb", "localhost")
	m.recordTransactionCommit(context.Background(), "testdb", "localhost")
	m.recordTransactionRollback(context.Background(), "testdb", "localhost")
}

func TestMetrics_SessionOpenClose_ViaDriverAndSession(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)
	ctx := context.Background()

	driverTracer := &DriverTracer{
		Driver:        &mockDriver{},
		tracer:        noopTracer(),
		metrics:       m,
		serverAddress: "localhost",
	}

	session := driverTracer.NewSession(ctx, neo4j.SessionConfig{})

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	metrics := collectMetricsByName(rm)

	sessionCount, ok := metrics["db.client.session.count"]
	require.True(t, ok)
	assertCounterValue(t, sessionCount, 1)

	sessionActive, ok := metrics["db.client.session.active"]
	require.True(t, ok)
	assertUpDownCounterValue(t, sessionActive, 1)

	// Close the session
	err := session.Close(ctx)
	require.NoError(t, err)

	rm = metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &rm))
	metrics = collectMetricsByName(rm)

	sessionActive, ok = metrics["db.client.session.active"]
	require.True(t, ok)
	assertUpDownCounterValue(t, sessionActive, 0)
}

func TestMetrics_TransactionCommit_ViaExplicitTransaction(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)
	ctx := context.Background()

	_, span := noopTracer().Start(ctx, "test")
	txTracer := NewExplicitTransactionTracer(ctx, &mockExplicitTransaction{}, span, noopTracer(),
		WithTransactionMetrics(m),
		WithTransactionServerAddress("localhost"),
		WithTransactionDBNamespace("testdb"),
	)

	err := txTracer.Commit(ctx)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	metrics := collectMetricsByName(rm)

	commitCount, ok := metrics["db.client.transaction.commit.count"]
	require.True(t, ok)
	assertCounterValue(t, commitCount, 1)
}

func TestMetrics_TransactionRollback_ViaExplicitTransaction(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m := newMetrics(mp)
	ctx := context.Background()

	_, span := noopTracer().Start(ctx, "test")
	txTracer := NewExplicitTransactionTracer(ctx, &mockExplicitTransaction{}, span, noopTracer(),
		WithTransactionMetrics(m),
		WithTransactionServerAddress("localhost"),
		WithTransactionDBNamespace("testdb"),
	)

	err := txTracer.Rollback(ctx)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	metrics := collectMetricsByName(rm)

	rollbackCount, ok := metrics["db.client.transaction.rollback.count"]
	require.True(t, ok)
	assertCounterValue(t, rollbackCount, 1)
}

// --- Test helpers ---

func noopTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer(tracerName)
}

// mockResultSummaryWithCounters implements neo4j.ResultSummary with configurable counters.
type mockResultSummaryWithCounters struct {
	neo4j.ResultSummary

	availableAfter time.Duration
	consumedAfter  time.Duration
	counters       neo4j.Counters
}

func (m *mockResultSummaryWithCounters) Server() neo4j.ServerInfo {
	return &mockServerInfo{}
}

func (m *mockResultSummaryWithCounters) Query() neo4j.Query {
	return nil
}

func (m *mockResultSummaryWithCounters) Counters() neo4j.Counters {
	return m.counters
}

func (m *mockResultSummaryWithCounters) ResultAvailableAfter() time.Duration {
	return m.availableAfter
}

func (m *mockResultSummaryWithCounters) ResultConsumedAfter() time.Duration {
	return m.consumedAfter
}

func (m *mockResultSummaryWithCounters) Database() neo4j.DatabaseInfo {
	return &mockDatabaseInfo{}
}

// mockCounters implements neo4j.Counters.
type mockCounters struct {
	nodesCreated         int
	nodesDeleted         int
	relationshipsCreated int
	relationshipsDeleted int
	propertiesSet        int
	labelsAdded          int
	labelsRemoved        int
	indexesAdded         int
	indexesRemoved       int
	constraintsAdded     int
	constraintsRemoved   int
	systemUpdates        int
}

func (c *mockCounters) ContainsUpdates() bool       { return true }
func (c *mockCounters) NodesCreated() int           { return c.nodesCreated }
func (c *mockCounters) NodesDeleted() int           { return c.nodesDeleted }
func (c *mockCounters) RelationshipsCreated() int   { return c.relationshipsCreated }
func (c *mockCounters) RelationshipsDeleted() int   { return c.relationshipsDeleted }
func (c *mockCounters) PropertiesSet() int          { return c.propertiesSet }
func (c *mockCounters) LabelsAdded() int            { return c.labelsAdded }
func (c *mockCounters) LabelsRemoved() int          { return c.labelsRemoved }
func (c *mockCounters) IndexesAdded() int           { return c.indexesAdded }
func (c *mockCounters) IndexesRemoved() int         { return c.indexesRemoved }
func (c *mockCounters) ConstraintsAdded() int       { return c.constraintsAdded }
func (c *mockCounters) ConstraintsRemoved() int     { return c.constraintsRemoved }
func (c *mockCounters) SystemUpdates() int          { return c.systemUpdates }
func (c *mockCounters) ContainsSystemUpdates() bool { return c.systemUpdates > 0 }

// collectMetricsByName collects all metrics from ResourceMetrics into a map by name.
func collectMetricsByName(rm metricdata.ResourceMetrics) map[string]metricdata.Metrics {
	result := make(map[string]metricdata.Metrics)

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			result[m.Name] = m
		}
	}

	return result
}

// assertCounterValue asserts the sum of all data points in a counter metric.
func assertCounterValue(t *testing.T, m metricdata.Metrics, expected int64) {
	t.Helper()

	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64] for metric %s, got %T", m.Name, m.Data)
	}

	var total int64

	for _, dp := range sum.DataPoints {
		total += dp.Value
	}

	assert.Equal(t, expected, total, "metric %s", m.Name)
}

// assertUpDownCounterValue asserts the sum of all data points in an up-down counter metric.
func assertUpDownCounterValue(t *testing.T, m metricdata.Metrics, expected int64) {
	t.Helper()

	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64] for metric %s, got %T", m.Name, m.Data)
	}

	var total int64

	for _, dp := range sum.DataPoints {
		total += dp.Value
	}

	assert.Equal(t, expected, total, "metric %s", m.Name)
}

// assertHasAttribute asserts that at least one data point in the metric has the given attribute.
func assertHasAttribute(t *testing.T, m metricdata.Metrics, key, value string) {
	t.Helper()

	found := false

	iterateDataPoints(m, func(attrs attribute.Set) {
		v, exists := attrs.Value(attribute.Key(key))
		if exists && v.AsString() == value {
			found = true
		}
	})

	assert.True(t, found, "expected attribute %s=%s in metric %s", key, value, m.Name)
}

// assertDoesNotHaveAttribute asserts that no data point has the given attribute key.
func assertDoesNotHaveAttribute(t *testing.T, m metricdata.Metrics, key string) {
	t.Helper()

	found := false

	iterateDataPoints(m, func(attrs attribute.Set) {
		_, exists := attrs.Value(attribute.Key(key))
		if exists {
			found = true
		}
	})

	assert.False(t, found, "did not expect attribute %s in metric %s", key, m.Name)
}

func iterateDataPoints(m metricdata.Metrics, fn func(attrs attribute.Set)) {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			fn(dp.Attributes)
		}
	case metricdata.Sum[float64]:
		for _, dp := range data.DataPoints {
			fn(dp.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, dp := range data.DataPoints {
			fn(dp.Attributes)
		}
	case metricdata.Histogram[int64]:
		for _, dp := range data.DataPoints {
			fn(dp.Attributes)
		}
	}
}
