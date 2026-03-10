# Neo4J Tracing

[//]: # (![Version]&#40;https://img.shields.io/github/v/tag/raito-io/neo4j-tracing?sort=semver&label=version&color=651FFF&#41;)

[//]: # ([![Build]&#40;https://img.shields.io/github/actions/workflow/status/raito-io/go-dynamo-utils/build.yml?branch=main&#41;]&#40;https://github.com/raito-io/go-dynamo-utils/actions/workflows/build.yml&#41;)

[//]: # ([![Contribute]&#40;https://img.shields.io/badge/Contribute-🙌-green.svg&#41;]&#40;/CONTRIBUTING.md&#41;)

[//]: # ([![Go version]&#40;https://img.shields.io/github/go-mod/go-version/raito-io/neo4j-tracing?color=7fd5ea&#41;]&#40;https://golang.org/&#41;)

[//]: # ([![Software License]&#40;https://img.shields.io/badge/license-Apache%202-brightgreen.svg?label=license&#41;]&#40;/LICENSE&#41;)

[//]: # ([![Go Reference]&#40;https://pkg.go.dev/badge/github.com/raito-io/neo4j-tracing.svg&#41;]&#40;https://pkg.go.dev/github.com/raito-io/neo4j-tracing&#41;)

## Introduction
`neo4jtracing` is a go library that enables otel distribute tracing for neo4j driver v5. 

## Getting Started
Add this library as a dependency via `go get github.com/collibra/go-neo4j-tracing`

## Enable tracing
Tracing can be enabled by using the `neo4j_tracing.Neo4jTracer` object. 
The `Neo4jTracer` a factory that creates `neo4j.DriverWithContext` objects that are wrapped so distributed tracing can be applied.

Start using tracing is very easy. A regular neo4j driver will be created as follows:
```go
package main

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
    dbUri := "neo4j://localhost" // scheme://host(:port) (default port is 7687)
    driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth("neo4j", "letmein!", ""))
    if err != nil {
        panic(err)
    }
    // Do something useful
}
```

To enable tracing you need to create your driver by using the `Neo4jTracer` object.
```go
package main

import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    neo4j_tracing "github.com/collibra/go-neo4j-tracing"
)

func main() {
    driverFactory := neo4j_tracing.NewNeo4jTracer()
	
    dbUri := "neo4j://localhost" // scheme://host(:port) (default port is 7687)
    driver, err := driverFactory.NewDriverWithContext(dbUri, neo4j.BasicAuth("neo4j", "letmein!", ""))
    if err != nil {
        panic(err)
    }
    // Do something useful
}
```

### Options
The following options could be used to customize the tracing and metrics behavior:
- `WithTracerProvider(provider)`: Specifies a custom tracer provider. By default, the global OpenTelemetry tracer provider is used.
- `WithMeterProvider(provider)`: Specifies a meter provider to use for recording metrics. If none is specified, metrics are not recorded.

Those options are passed as argument to the `neo4j_tracing.NewNeo4jTracer()` function.

## Enable metrics
Metrics can be enabled by passing a `WithMeterProvider` option to the `NewNeo4jTracer` function:

```go
package main

import (
    "github.com/neo4j/neo4j-go-driver/v6/neo4j"
    neo4j_tracing "github.com/collibra/go-neo4j-tracing"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Set up your meter provider (e.g., with an OTLP exporter)
    mp := sdkmetric.NewMeterProvider()
    defer mp.Shutdown(context.Background())

    driverFactory := neo4j_tracing.NewNeo4jTracer(
        neo4j_tracing.WithMeterProvider(mp),
    )

    dbUri := "neo4j://localhost"
    driver, err := driverFactory.NewDriver(dbUri, neo4j.BasicAuth("neo4j", "letmein!", ""))
    if err != nil {
        panic(err)
    }
    // Do something useful
}
```

### Available metrics

#### Core metrics (recorded for every operation)

| Metric | Type | Description |
|--------|------|-------------|
| `db.client.operation.duration` | Float64Histogram (seconds) | Client-side duration of each operation |
| `db.client.operation.count` | Int64Counter | Total number of operations executed |
| `db.client.error.count` | Int64Counter | Total number of failed operations |

Common attributes: `db.system.name="neo4j"`, `db.operation.name`, `db.namespace`, `server.address`, `error.type` (on failure).

#### ResultSummary metrics (recorded on `Consume()` / `Single()` calls)

| Metric | Type | Description |
|--------|------|-------------|
| `db.client.result.available_after` | Float64Histogram (seconds) | Server-side time until result was available |
| `db.client.result.consumed_after` | Float64Histogram (seconds) | Server-side time to consume result |
| `db.client.result.nodes_created` | Int64Counter | Cumulative nodes created |
| `db.client.result.nodes_deleted` | Int64Counter | Cumulative nodes deleted |
| `db.client.result.relationships_created` | Int64Counter | Cumulative relationships created |
| `db.client.result.relationships_deleted` | Int64Counter | Cumulative relationships deleted |
