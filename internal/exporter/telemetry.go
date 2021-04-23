package exporter

import (
	"context"
	"crypto/tls"

	"github.com/newrelic/newrelic-pixie-integration/internal/errors"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type openTelemetry struct {
	ctx           context.Context
	metricsClient colmetricpb.MetricsServiceClient
	traceClient   coltracepb.TraceServiceClient
}

func (e *openTelemetry) SendMetrics(metrics []*metricpb.ResourceMetrics) errors.Error {
	_, err := e.metricsClient.Export(e.ctx, &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: metrics,
	})
	if err != nil {
		return errors.ExporterError(err.Error())
	}
	return nil
}

func (e *openTelemetry) SendSpans(spans []*tracepb.ResourceSpans) errors.Error {
	_, err := e.traceClient.Export(e.ctx, &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: spans,
	})
	if err != nil {
		return errors.ExporterError(err.Error())
	}
	return nil
}

func createConnection(ctx context.Context, endpoint string, apiKey string) (*grpc.ClientConn, context.Context, errors.Error) {
	outgoingCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", apiKey))
	var opts []grpc.DialOption
	tlsDialOption := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	opts = append(opts, tlsDialOption)

	conn, err := grpc.DialContext(outgoingCtx, endpoint, opts...)
	if err != nil {
		return nil, context.Background(), errors.ConnectionError(err.Error())
	}

	return conn, outgoingCtx, nil
}
