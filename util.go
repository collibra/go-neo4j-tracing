package neo4j_tracing

const tracerName = "github.com/collibra/go-neo4j-tracing"
const serviceID = "neo4j"

func spanName(operation string) string {
	return serviceID + "." + operation
}
