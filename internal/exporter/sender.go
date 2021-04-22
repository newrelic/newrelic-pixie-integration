package exporter

import (
	"context"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type Exporter interface {
	SendMetrics(metrics []*metricpb.ResourceMetrics) error
	SendSpans(spans []*tracepb.ResourceSpans) error
}

func New(ctx context.Context, endpoint string, apiKey string) (Exporter, error) {
	conn, outgoingCtx, err := createConnection(ctx, endpoint, apiKey)
	if err != nil {
		return nil, err
	}
	return &openTelemetry{
		ctx:           outgoingCtx,
		metricsClient: colmetricpb.NewMetricsServiceClient(conn),
		traceClient:   coltracepb.NewTraceServiceClient(conn),
	}, nil
}
