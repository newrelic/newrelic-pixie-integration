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

func JVM(clusterName, clusterID string, collectIntervalSec int64) MetricsAdapter {
	return newJvm(clusterName, clusterID, collectIntervalSec)
}

func HTTPMetrics(clusterName, clusterID string, collectIntervalSec int64) MetricsAdapter {
	return newHttpMetrics(clusterName, clusterID, collectIntervalSec)
}

func HTTPSpans(clusterName, clusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newHttpSpans(clusterName, clusterID, collectIntervalSec, spanLimit)
}

func MySQL(clusterName, clusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newMysql(clusterName, clusterID, collectIntervalSec, spanLimit)
}

func PgSQL(clusterName, clusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newPogsql(clusterName, clusterID, collectIntervalSec, spanLimit)
}
