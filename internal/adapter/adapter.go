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
	CollectIntervalSec() int64
	Adapt(r *types.Record) ([]*metricpb.ResourceMetrics, error)
}

type SpansAdapter interface {
	ID() string
	Script() string
	CollectIntervalSec() int64
	Adapt(r *types.Record) ([]*tracepb.ResourceSpans, error)
}

func JVM(clusterName string, collectIntervalSec int64) MetricsAdapter {
	return newJvm(clusterName, collectIntervalSec)
}

func HTTPMetrics(clusterName string, collectIntervalSec int64) MetricsAdapter {
	return newHttpMetrics(clusterName, collectIntervalSec)
}

func HTTPSpans(clusterName string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newHttpSpans(clusterName, collectIntervalSec, spanLimit)
}

func MySQL(clusterName string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newMysql(clusterName, collectIntervalSec, spanLimit)
}

func PgSQL(clusterName string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newPogsql(clusterName, collectIntervalSec, spanLimit)
}
