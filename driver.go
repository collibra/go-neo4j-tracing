package neo4j_tracing

import (
	"context"
	"net/url"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j/auth"
	config2 "github.com/neo4j/neo4j-go-driver/v6/neo4j/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Neo4jTracer wraps a neo4j.Tracer object so the calls can be traced with open telemetry distributed tracing
type Neo4jTracer struct {
	tracer  trace.Tracer
	metrics *metrics
}

// NewNeo4jTracer creates an object that will wrap neo4j drivers with a tracing object
func NewNeo4jTracer(opts ...Option) *Neo4jTracer {
	cfg := config{}
	for _, o := range opts {
		o.apply(&cfg)
	}

	if cfg.TraceProvider == nil {
		cfg.TraceProvider = otel.GetTracerProvider()
	}

	return &Neo4jTracer{
		tracer:  cfg.TraceProvider.Tracer(tracerName),
		metrics: newMetrics(cfg.MeterProvider),
	}
}

// NewDriverWithContext is the entry point to the neo4j driver to create an instance of a neo4j.DriverWithContext that is wrapped by a tracing object
// More information about the arguments can be found in the underlying neo4j driver call neo4j.NewDriverWithContext
func (t *Neo4jTracer) NewDriver(target string, auth auth.TokenManager, configurers ...func(config2 *config2.Config)) (_ neo4j.Driver, err error) { //nolint:staticcheck
	driver, err := neo4j.NewDriver(target, auth, configurers...)
	if err != nil {
		return nil, err
	}

	serverAddress := parseServerAddress(target)

	return &DriverTracer{
		Driver:        driver,
		tracer:        t.tracer,
		metrics:       t.metrics,
		serverAddress: serverAddress,
	}, nil
}

type DriverTracer struct {
	neo4j.Driver

	tracer        trace.Tracer
	metrics       *metrics
	serverAddress string
}

// NewSession calls neo4j.DriverWithContext.NewSession and wraps the resulting neo4j.SessionWithContext with a tracing object
func (n *DriverTracer) NewSession(ctx context.Context, config neo4j.SessionConfig) neo4j.Session {
	bookmarks := make([]neo4j.Bookmarks, 0, 2)

	if config.Bookmarks != nil {
		bookmarks = append(bookmarks, config.Bookmarks)
	}

	if config.BookmarkManager != nil {
		b, err := config.BookmarkManager.GetBookmarks(ctx)
		if err == nil {
			bookmarks = append(bookmarks, b)
		}
	}

	return &SessionTracer{
		Session: n.Driver.NewSession(ctx, config),
		tracer:  n.tracer,
		metrics: n.metrics,
		attributes: SessionAttributes{
			DatabaseName: config.DatabaseName,
			AccessMode:   config.AccessMode,
			Bookmarks:    neo4j.CombineBookmarks(bookmarks...),
			FetchSize:    config.FetchSize,
		},
		serverAddress: n.serverAddress,
	}
}

// VerifyConnectivity calls neo4j.DriverWithContext.VerifyConnectivity and trace the call
func (n *DriverTracer) VerifyConnectivity(ctx context.Context) (err error) {
	start := time.Now()
	spanCtx, span := n.tracer.Start(ctx, spanName("VerifyConnectivity"), trace.WithSpanKind(trace.SpanKindClient))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
		n.metrics.recordOperation(ctx, start, "VerifyConnectivity", "", n.serverAddress, err)
	}()

	return n.Driver.VerifyConnectivity(spanCtx)
}

// VerifyAuthentication calls neo4j.DriverWithContext.VerifyAuthentication and trace the call
func (n *DriverTracer) VerifyAuthentication(ctx context.Context, auth *neo4j.AuthToken) (err error) {
	start := time.Now()
	spanCtx, span := n.tracer.Start(ctx, spanName("VerifyAuthentication"), trace.WithSpanKind(trace.SpanKindClient))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
		n.metrics.recordOperation(ctx, start, "VerifyAuthentication", "", n.serverAddress, err)
	}()

	return n.Driver.VerifyAuthentication(spanCtx, auth)
}

// GetServerInfo calls neo4j.GetServerInfo.VerifyConnectivity and trace the call
func (n *DriverTracer) GetServerInfo(ctx context.Context) (_ neo4j.ServerInfo, err error) {
	start := time.Now()
	spanCtx, span := n.tracer.Start(ctx, spanName("GetServerInfo"), trace.WithSpanKind(trace.SpanKindClient))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
		n.metrics.recordOperation(ctx, start, "GetServerInfo", "", n.serverAddress, err)
	}()

	return n.Driver.GetServerInfo(spanCtx)
}

func parseServerAddress(target string) string {
	u, err := url.Parse(target)
	if err != nil {
		return target
	}

	return u.Hostname()
}
