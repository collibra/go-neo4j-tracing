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

func TestNewResultTracer(t *testing.T) {
	t.Run("should create a new result tracer", func(t *testing.T) {
		tracer := noop.NewTracerProvider().Tracer(tracerName)
		resultTracer := NewResultTracer(t.Context(), &mockResult{}, tracer)
		assert.NotNil(t, resultTracer)
	})
}

func TestResultTracer_NextRecord(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() neo4j.Result
		expected  bool
	}{
		{
			name: "should trace call and return true",
			setupMock: func() neo4j.Result {
				return &mockResult{
					nextRecordFunc: func(ctx context.Context, record **neo4j.Record) bool {
						return true
					},
				}
			},
			expected: true,
		},
		{
			name: "should trace call and return false",
			setupMock: func() neo4j.Result {
				return &mockResult{
					nextRecordFunc: func(ctx context.Context, record **neo4j.Record) bool {
						return false
					},
				}
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			var record *neo4j.Record
			res := resultTracer.NextRecord(t.Context(), &record)

			assert.Equal(t, tt.expected, res)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.NextRecord", spans[0].Name())
		})
	}
}

func TestResultTracer_Next(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() neo4j.Result
		expected  bool
	}{
		{
			name: "should trace call and return true",
			setupMock: func() neo4j.Result {
				return &mockResult{
					nextFunc: func(ctx context.Context) bool {
						return true
					},
				}
			},
			expected: true,
		},
		{
			name: "should trace call and return false",
			setupMock: func() neo4j.Result {
				return &mockResult{
					nextFunc: func(ctx context.Context) bool {
						return false
					},
				}
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			res := resultTracer.Next(t.Context())

			assert.Equal(t, tt.expected, res)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.Next", spans[0].Name())
		})
	}
}

func TestResultTracer_PeekRecord(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() neo4j.Result
		expected  bool
	}{
		{
			name: "should trace call and return true",
			setupMock: func() neo4j.Result {
				return &mockResult{
					peekRecordFunc: func(ctx context.Context, record **neo4j.Record) bool {
						return true
					},
				}
			},
			expected: true,
		},
		{
			name: "should trace call and return false",
			setupMock: func() neo4j.Result {
				return &mockResult{
					peekRecordFunc: func(ctx context.Context, record **neo4j.Record) bool {
						return false
					},
				}
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			var record *neo4j.Record
			res := resultTracer.PeekRecord(t.Context(), &record)

			assert.Equal(t, tt.expected, res)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.PeekRecord", spans[0].Name())
		})
	}
}

func TestResultTracer_Peek(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() neo4j.Result
		expected  bool
	}{
		{
			name: "should trace call and return true",
			setupMock: func() neo4j.Result {
				return &mockResult{
					peekFunc: func(ctx context.Context) bool {
						return true
					},
				}
			},
			expected: true,
		},
		{
			name: "should trace call and return false",
			setupMock: func() neo4j.Result {
				return &mockResult{
					peekFunc: func(ctx context.Context) bool {
						return false
					},
				}
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			res := resultTracer.Peek(t.Context())

			assert.Equal(t, tt.expected, res)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.Peek", spans[0].Name())
		})
	}
}

func TestResultTracer_Collect(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.Result
		expectedError error
	}{
		{
			name: "should trace call and return records",
			setupMock: func() neo4j.Result {
				return &mockResult{
					collectFunc: func(ctx context.Context) ([]*neo4j.Record, error) {
						return []*neo4j.Record{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "should trace call and return error",
			setupMock: func() neo4j.Result {
				return &mockResult{
					collectFunc: func(ctx context.Context) ([]*neo4j.Record, error) {
						return nil, errors.New("collect error")
					},
				}
			},
			expectedError: errors.New("collect error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			_, err := resultTracer.Collect(t.Context())

			assert.Equal(t, tt.expectedError, err)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.Collect", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestResultTracer_Single(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.Result
		expectedError error
	}{
		{
			name: "should trace call and return single record",
			setupMock: func() neo4j.Result {
				return &mockResult{
					singleFunc: func(ctx context.Context) (*neo4j.Record, error) {
						return &neo4j.Record{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "should trace call and return error",
			setupMock: func() neo4j.Result {
				return &mockResult{
					singleFunc: func(ctx context.Context) (*neo4j.Record, error) {
						return nil, errors.New("single error")
					},
				}
			},
			expectedError: errors.New("single error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			_, err := resultTracer.Single(t.Context())

			assert.Equal(t, tt.expectedError, err)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.Single", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestResultTracer_Consume(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() neo4j.Result
		expectedError error
	}{
		{
			name: "should trace call and return summary",
			setupMock: func() neo4j.Result {
				return &mockResult{
					consumeFunc: func(ctx context.Context) (neo4j.ResultSummary, error) {
						return &mockResultSummary{}, nil
					},
				}
			},
			expectedError: nil,
		},
		{
			name: "should trace call and return error",
			setupMock: func() neo4j.Result {
				return &mockResult{
					consumeFunc: func(ctx context.Context) (neo4j.ResultSummary, error) {
						return nil, errors.New("consume error")
					},
				}
			},
			expectedError: errors.New("consume error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			resultTracer := NewResultTracer(t.Context(), tt.setupMock(), tracer)
			_, err := resultTracer.Consume(t.Context())

			assert.Equal(t, tt.expectedError, err)
			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.Record.Consume", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}
