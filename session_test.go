package neo4j_tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
)

func TestSessionAttributes_SetAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes SessionAttributes
		expected   []attribute.KeyValue
	}{
		{
			name: "read access mode",
			attributes: SessionAttributes{
				AccessMode:   neo4j.AccessModeRead,
				Bookmarks:    neo4j.Bookmarks{"b1", "b2"},
				DatabaseName: "testdb",
				FetchSize:    100,
			},
			expected: []attribute.KeyValue{
				attribute.StringSlice(bookmarksStartAttrKey, []string{"b1", "b2"}),
				semconv.DBName("testdb"),
				attribute.Int(fetchAttrKey, 100),
				attribute.String(accessModeAttrKey, "READ"),
			},
		},
		{
			name: "write access mode",
			attributes: SessionAttributes{
				AccessMode:   neo4j.AccessModeWrite,
				Bookmarks:    neo4j.Bookmarks{"b3"},
				DatabaseName: "writedb",
				FetchSize:    -1,
			},
			expected: []attribute.KeyValue{
				attribute.StringSlice(bookmarksStartAttrKey, []string{"b3"}),
				semconv.DBName("writedb"),
				attribute.Int(fetchAttrKey, -1),
				attribute.String(accessModeAttrKey, "WRITE"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			_, span := provider.Tracer(tracerName).Start(t.Context(), "test")

			tt.attributes.SetAttributes(span)
			span.End()

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.ElementsMatch(t, tt.expected, spans[0].Attributes())
		})
	}
}

func TestSessionTracer_BeginTransaction(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.Session
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.Session {
				return &mockSession{
					beginTransactionFunc: func(ctx context.Context, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
						return &mockExplicitTransaction{}, nil
					},
					lastBookmarksFunc: func() neo4j.Bookmarks {
						return neo4j.Bookmarks{"new-bookmark"}
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.Session {
				return &mockSession{
					beginTransactionFunc: func(ctx context.Context, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
						return nil, errors.New("tx error")
					},
					lastBookmarksFunc: func() neo4j.Bookmarks {
						return neo4j.Bookmarks{"new-bookmark"}
					},
				}
			},
			expectedError: errors.New("tx error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			sessionTracer := &SessionTracer{
				Session: tt.setupMock(),
				tracer:  tracer,
				attributes: SessionAttributes{
					Bookmarks: neo4j.Bookmarks{"start-bookmark"},
				},
			}

			tx, err := sessionTracer.BeginTransaction(t.Context())

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, tx)
				spans := sr.Ended()
				assert.Len(t, spans, 1)
				assert.Equal(t, "Session.BeginTransaction", spans[0].Name())
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tx)
				// The span is ended by tx.Close(), so we can't check it here.
				// We'll test that in the transaction tests.
				tx.Close(t.Context())
				spans := sr.Ended()
				assert.Len(t, spans, 1)
				assert.Equal(t, "Session.BeginTransaction", spans[0].Name())
				assert.Contains(t, spans[0].Attributes(), attribute.StringSlice(bookmarksEndAttrKey, []string{"new-bookmark"}))
			}
		})
	}
}

func TestSessionTracer_ExecuteRead(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(work neo4j.ManagedTransactionWork) neo4j.Session
		expectedError error
		expectedValue any
	}{
		{
			name: "success",
			setupMock: func(work neo4j.ManagedTransactionWork) neo4j.Session {
				return &mockSession{
					executeReadFunc: func(ctx context.Context, actualWork neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
						return actualWork(&mockManagedTransaction{})
					},
				}
			},
			expectedError: nil,
			expectedValue: "read-result",
		},
		{
			name: "failure",
			setupMock: func(work neo4j.ManagedTransactionWork) neo4j.Session {
				return &mockSession{
					executeReadFunc: func(ctx context.Context, actualWork neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
						return nil, errors.New("read error")
					},
				}
			},
			expectedError: errors.New("read error"),
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			work := func(tx neo4j.ManagedTransaction) (any, error) {
				assert.IsType(t, &ManagedTransactionTracer{}, tx)
				return "read-result", nil
			}

			sessionTracer := &SessionTracer{
				Session: tt.setupMock(work),
				tracer:  tracer,
			}

			val, err := sessionTracer.ExecuteRead(t.Context(), work)

			assert.Equal(t, tt.expectedError, err)
			if tt.expectedError == nil {
				assert.Equal(t, tt.expectedValue, val)
			}
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.ExecuteRead", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestSessionTracer_ExecuteWrite(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(work neo4j.ManagedTransactionWork) neo4j.Session
		expectedError error
		expectedValue any
	}{
		{
			name: "success",
			setupMock: func(work neo4j.ManagedTransactionWork) neo4j.Session {
				return &mockSession{
					executeWriteFunc: func(ctx context.Context, actualWork neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
						return actualWork(&mockManagedTransaction{})
					},
				}
			},
			expectedError: nil,
			expectedValue: "write-result",
		},
		{
			name: "failure",
			setupMock: func(work neo4j.ManagedTransactionWork) neo4j.Session {
				return &mockSession{
					executeWriteFunc: func(ctx context.Context, actualWork neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
						return nil, errors.New("write error")
					},
				}
			},
			expectedError: errors.New("write error"),
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			work := func(tx neo4j.ManagedTransaction) (any, error) {
				assert.IsType(t, &ManagedTransactionTracer{}, tx)
				return "write-result", nil
			}

			sessionTracer := &SessionTracer{
				Session: tt.setupMock(work),
				tracer:  tracer,
			}

			val, err := sessionTracer.ExecuteWrite(t.Context(), work)

			assert.Equal(t, tt.expectedError, err)
			if tt.expectedError == nil {
				assert.Equal(t, tt.expectedValue, val)
			}
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.ExecuteWrite", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestSessionTracer_Run(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.Session
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.Session {
				return &mockSession{
					runFunc: func(ctx context.Context, cypher string, params map[string]any, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.Result, error) {
						return &mockResult{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.Session {
				return &mockSession{
					runFunc: func(ctx context.Context, cypher string, params map[string]any, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.Result, error) {
						return nil, errors.New("run error")
					},
				}
			},
			expectedError: errors.New("run error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			sessionTracer := &SessionTracer{
				Session: tt.setupMock(),
				tracer:  tracer,
			}

			result, err := sessionTracer.Run(t.Context(), "RETURN 1", nil)

			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Run", spans[0].Name())

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			} else {
				assert.NotNil(t, result)
				assert.IsType(t, &ResultTracer{}, result)
			}
		})
	}
}
