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

func JVM(clusterName string) MetricsAdapter {
	return &jvm{clusterName}
}

func HTTPMetrics(clusterName string) MetricsAdapter {
	return &httpMetrics{clusterName}
}

func HTTPSpans(clusterName string) SpansAdapter {
	return &httpSpans{clusterName}
}

func MySQL(clusterName string) SpansAdapter {
	return &mysql{clusterName}
}

func PgSQL(clusterName string) SpansAdapter {
	return &pogsql{clusterName}
}
