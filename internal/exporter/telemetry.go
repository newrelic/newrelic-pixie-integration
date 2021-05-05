package exporter

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/newrelic/infrastructure-agent/pkg/log"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const chunkSize = 1000

type openTelemetry struct {
	ctx           context.Context
	metricsClient colmetricpb.MetricsServiceClient
	traceClient   coltracepb.TraceServiceClient
}

func (e *openTelemetry) SendMetrics(metrics []*metricpb.ResourceMetrics) int {
	lenMetrics := len(metrics)
	if lenMetrics == 0 {
		return 0
	}
	startTime := time.Now()
	processed := 0
	var wg sync.WaitGroup
	for i := 0; i < lenMetrics; i += chunkSize {
		end := i + chunkSize
		if end > lenMetrics {
			end = lenMetrics
		}
		metricsBatch := metrics[i:end]
		wg.Add(1)
		go func(batch []*metricpb.ResourceMetrics) {
			_, err := e.metricsClient.Export(e.ctx, &colmetricpb.ExportMetricsServiceRequest{
				ResourceMetrics: batch,
			})
			if err != nil {
				log.Errorf("missing %d metrics. error while sending: %s", len(batch), err)
			} else {
				processed += len(batch)
			}
			wg.Done()
		}(metricsBatch)
	}
	wg.Wait()
	log.Debugf("It took %v to send %d metrics", time.Since(startTime).Seconds(), processed)
	return processed
}

func (e *openTelemetry) SendSpans(spans []*tracepb.ResourceSpans) int {
	lenSpans := len(spans)
	if lenSpans == 0 {
		return 0
	}
	startTime := time.Now()
	processed := 0
	var wg sync.WaitGroup
	for i := 0; i < lenSpans; i += chunkSize {
		end := i + chunkSize
		if end > lenSpans {
			end = lenSpans
		}
		spansBatch := spans[i:end]
		wg.Add(1)
		go func(batch []*tracepb.ResourceSpans) {
			_, err := e.traceClient.Export(e.ctx, &coltracepb.ExportTraceServiceRequest{
				ResourceSpans: batch,
			})
			if err != nil {
				log.Errorf("missing %d spans. error while sending: %s", len(batch), err)
			} else {
				processed += len(batch)
			}
			wg.Done()
		}(spansBatch)
	}
	wg.Wait()
	log.Debugf("It took %v to send %d spans", time.Since(startTime).Seconds(), processed)
	return processed
}

func createConnection(ctx context.Context, endpoint string, apiKey string) (*grpc.ClientConn, context.Context, error) {
	outgoingCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", apiKey))
	var opts []grpc.DialOption
	tlsDialOption := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	opts = append(opts, tlsDialOption)
	conn, err := grpc.DialContext(outgoingCtx, endpoint, opts...)
	if err != nil {
		return nil, context.Background(), fmt.Errorf("error creating grpc connection: %w", err)
	}

	return conn, outgoingCtx, nil
}
