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
	Adapt(rh *ResourceHelper, r *types.Record) ([]*metricpb.ResourceMetrics, error)
}

type SpansAdapter interface {
	ID() string
	Script() string
	CollectIntervalSec() int64
	Adapt(rh *ResourceHelper, r *types.Record) ([]*tracepb.ResourceSpans, error)
}

func JVM(clusterName, pixieClusterID string, collectIntervalSec int64) MetricsAdapter {
	return newJvm(clusterName, pixieClusterID, collectIntervalSec)
}

func HTTPMetrics(clusterName, pixieClusterID string, collectIntervalSec int64) MetricsAdapter {
	return newHttpMetrics(clusterName, pixieClusterID, collectIntervalSec)
}

func HTTPSpans(clusterName, pixieClusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newHttpSpans(clusterName, pixieClusterID, collectIntervalSec, spanLimit)
}

func MySQL(clusterName, pixieClusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newMysql(clusterName, pixieClusterID, collectIntervalSec, spanLimit)
}

func PgSQL(clusterName, pixieClusterID string, collectIntervalSec, spanLimit int64) SpansAdapter {
	return newPogsql(clusterName, pixieClusterID, collectIntervalSec, spanLimit)
}
