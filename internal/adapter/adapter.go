package adapter

import (
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi/types"
)

const instrumentationName = "pixie"

var (
	idGenerator            = defaultIDGenerator()
	instrumentationLibrary = &commonpb.InstrumentationLibrary{
		Name:    instrumentationName,
		Version: "1.0.0",
	}
)

type MetricsAdapter interface {
	ID() string
	Script() string
	Adapt(r *types.Record) ([]*metricpb.ResourceMetrics, error)
}

type SpansAdapter interface {
	ID() string
	Script() string
	Adapt(r *types.Record) ([]*tracepb.ResourceSpans, error)
}

var (
	JVM         = &jvm{}
	HTTPMetrics = &httpMetrics{}
	HTTPSpans   = &httpSpans{}
	MySQL       = &mysql{}
	PgSQL       = &pogsql{}
)
