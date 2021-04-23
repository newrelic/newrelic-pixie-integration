package exporter

import (
	"context"

	"github.com/newrelic/newrelic-pixie-integration/internal/config"
	"github.com/newrelic/newrelic-pixie-integration/internal/errors"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type Exporter interface {
	SendMetrics(metrics []*metricpb.ResourceMetrics) errors.Error
	SendSpans(spans []*tracepb.ResourceSpans) errors.Error
}

func New(ctx context.Context, cfg config.Exporter) (Exporter, errors.Error) {
	conn, outgoingCtx, err := createConnection(ctx, cfg.Endpoint(), cfg.LicenseKey())
	if err != nil {
		return nil, err
	}
	return &openTelemetry{
		ctx:           outgoingCtx,
		metricsClient: colmetricpb.NewMetricsServiceClient(conn),
		traceClient:   coltracepb.NewTraceServiceClient(conn),
	}, nil
}
