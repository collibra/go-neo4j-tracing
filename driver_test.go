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

func TestNewNeo4jTracer(t *testing.T) {
	t.Run("should create a new tracer with the default trace provider", func(t *testing.T) {
		tracer := NewNeo4jTracer()
		assert.NotNil(t, tracer)
		assert.NotNil(t, tracer.tracer)
	})

	t.Run("should create a new tracer with a custom trace provider", func(t *testing.T) {
		provider := noop.NewTracerProvider()
		tracer := NewNeo4jTracer(WithTracerProvider(provider))
		assert.NotNil(t, tracer)
		assert.Equal(t, provider.Tracer(tracerName), tracer.tracer)
	})
}

func TestDriverTracer_NewSession(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := provider.Tracer(tracerName)

	driverTracer := &DriverTracer{
		Driver: &mockDriver{
			newSessionFunc: func(ctx context.Context, config neo4j.SessionConfig) neo4j.Session {
				return &mockSession{}
			},
		},
		tracer: tracer,
	}

	tests := []struct {
		name string
		config neo4j.SessionConfig
		expectedDBName string
	}{
		{
			name: "with database name",
			config: neo4j.SessionConfig{DatabaseName: "testdb"},
			expectedDBName: "testdb",
		},
		{
			name: "without database name",
			config: neo4j.SessionConfig{},
			expectedDBName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := driverTracer.NewSession(t.Context(), tt.config)
			sessionTracer, ok := session.(*SessionTracer)
			assert.True(t, ok)
			assert.NotNil(t, sessionTracer)
			assert.Equal(t, tt.expectedDBName, sessionTracer.attributes.DatabaseName)
		})
	}
}

func TestDriverTracer_VerifyConnectivity(t *testing.T) {
	tests := []struct {
		name          string
		expectedError error
		setupMock     func() neo4j.Driver
	}{
		{
			name:          "success",
			expectedError: nil,
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					verifyConnectivityFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
		},
		{
			name:          "failure",
			expectedError: errors.New("some error"),
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					verifyConnectivityFunc: func(ctx context.Context) error {
						return errors.New("some error")
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			driverTracer := &DriverTracer{
				Driver: tt.setupMock(),
				tracer: tracer,
			}

			err := driverTracer.VerifyConnectivity(t.Context())
			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.VerifyConnectivity", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestDriverTracer_VerifyAuthentication(t *testing.T) {
	tests := []struct {
		name          string
		expectedError error
		setupMock     func() neo4j.Driver
	}{
		{
			name:          "success",
			expectedError: nil,
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					verifyAuthenticationFunc: func(ctx context.Context, auth *neo4j.AuthToken) error {
						return nil
					},
				}
			},
		},
		{
			name:          "failure",
			expectedError: errors.New("auth error"),
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					verifyAuthenticationFunc: func(ctx context.Context, auth *neo4j.AuthToken) error {
						return errors.New("auth error")
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			driverTracer := &DriverTracer{
				Driver: tt.setupMock(),
				tracer: tracer,
			}

			err := driverTracer.VerifyAuthentication(t.Context(), nil)
			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.VerifyAuthentication", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}

func TestDriverTracer_GetServerInfo(t *testing.T) {
	tests := []struct {
		name          string
		expectedError error
		setupMock     func() neo4j.Driver
	}{
		{
			name:          "success",
			expectedError: nil,
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					getServerInfoFunc: func(ctx context.Context) (neo4j.ServerInfo, error) {
						return &mockServerInfo{}, nil
					},
				}
			},
		},
		{
			name:          "failure",
			expectedError: errors.New("server info error"),
			setupMock: func() neo4j.Driver {
				return &mockDriver{
					getServerInfoFunc: func(ctx context.Context) (neo4j.ServerInfo, error) {
						return nil, errors.New("server info error")
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer(tracerName)

			driverTracer := &DriverTracer{
				Driver: tt.setupMock(),
				tracer: tracer,
			}

			_, err := driverTracer.GetServerInfo(t.Context())
			assert.Equal(t, tt.expectedError, err)

			spans := sr.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, "neo4j.GetServerInfo", spans[0].Name())
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), spans[0].Events()[0].Attributes[1].Value.AsString())
			}
		})
	}
}
