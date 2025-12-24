package neo4j_tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewManagedTransactionTracer(t *testing.T) {
	t.Run("should create a new managed transaction tracer", func(t *testing.T) {
		tracer := noop.NewTracerProvider().Tracer(tracerName)
		txTracer := NewManagedTransactionTracer(t.Context(), &mockManagedTransaction{}, tracer)
		assert.NotNil(t, txTracer)
	})
}

func TestManagedTransactionTracer_Run(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.ManagedTransaction
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.ManagedTransaction {
				return &mockManagedTransaction{
					runFunc: func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
						return &mockResult{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.ManagedTransaction {
				return &mockManagedTransaction{
					runFunc: func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
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

			txTracer := NewManagedTransactionTracer(t.Context(), tt.setupMock(), tracer)
			result, err := txTracer.Run(t.Context(), "RETURN 1", nil)

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

func TestNewExplicitTransactionTracer(t *testing.T) {
	t.Run("should create a new explicit transaction tracer", func(t *testing.T) {
		tracer := noop.NewTracerProvider().Tracer(tracerName)
		_, span := tracer.Start(t.Context(), "test")
		txTracer := NewExplicitTransactionTracer(t.Context(), &mockExplicitTransaction{}, span, tracer)
		assert.NotNil(t, txTracer)
	})
}

func TestExplicitTransactionTracer_Run(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.ExplicitTransaction
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					runFunc: func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
						return &mockResult{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					runFunc: func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
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
			_, span := tracer.Start(t.Context(), "test")

			txTracer := NewExplicitTransactionTracer(t.Context(), tt.setupMock(), span, tracer)
			result, err := txTracer.Run(t.Context(), "RETURN 1", nil)

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

func TestExplicitTransactionTracer_Commit(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.ExplicitTransaction
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					commitFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					commitFunc: func(ctx context.Context) error {
						return errors.New("commit error")
					},
				}
			},
			expectedError: errors.New("commit error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)
			_, span := tracer.Start(t.Context(), "test")

			txTracer := NewExplicitTransactionTracer(t.Context(), tt.setupMock(), span, tracer)
			err := txTracer.Commit(t.Context())

			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Commit", spans[0].Name())

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestExplicitTransactionTracer_Rollback(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.ExplicitTransaction
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					rollbackFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					rollbackFunc: func(ctx context.Context) error {
						return errors.New("rollback error")
					},
				}
			},
			expectedError: errors.New("rollback error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)
			_, span := tracer.Start(t.Context(), "test")

			txTracer := NewExplicitTransactionTracer(t.Context(), tt.setupMock(), span, tracer)
			err := txTracer.Rollback(t.Context())

			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Rollback", spans[0].Name())

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestExplicitTransactionTracer_Close(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.ExplicitTransaction
		expectedError error
	}{
		{
			name: "success",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					closeFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "failure",
			setupMock: func() neo4j.ExplicitTransaction {
				return &mockExplicitTransaction{
					closeFunc: func(ctx context.Context) error {
						return errors.New("close error")
					},
				}
			},
			expectedError: errors.New("close error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)
			_, span := tracer.Start(t.Context(), "test")

			txTracer := NewExplicitTransactionTracer(t.Context(), tt.setupMock(), span, tracer)
			err := txTracer.Close(t.Context())

			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "test", spans[0].Name())

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}
