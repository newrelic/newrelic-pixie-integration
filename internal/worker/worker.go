package worker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi"
)

type Worker interface {
	Spans(vadapter adapter.SpansAdapter)
	Metrics(adapter adapter.MetricsAdapter)
}

type worker struct {
	ctx         context.Context
	clusterName string
	vz          *pxapi.VizierClient
	exporter    exporter.Exporter
}

func Build(ctx context.Context, clusterName string, vz *pxapi.VizierClient, exporter exporter.Exporter) Worker {
	return &worker{
		ctx:         ctx,
		clusterName: clusterName,
		vz:          vz,
		exporter:    exporter,
	}
}

func (w *worker) Metrics(adapter adapter.MetricsAdapter) {
	h := &metricsHandler{
		handler: &handler{
			recordsHandled: 0,
		},
		adapter: adapter,
		metrics: make([]*metricpb.ResourceMetrics, 0),
	}
	run(w.ctx, adapter.ID(), adapter.Script(), w.vz, h, w.exporter)
}

func (w *worker) Spans(adapter adapter.SpansAdapter) {
	h := &spansHandler{
		handler: &handler{
			recordsHandled: 0,
		},
		adapter: adapter,
		spans:   make([]*tracepb.ResourceSpans, 0),
	}
	run(w.ctx, adapter.ID(), adapter.Script(), w.vz, h, w.exporter)
}

func run(ctx context.Context, name string, script string, vz *pxapi.VizierClient, h customHandler, exporter exporter.Exporter) {
	rm := &ResultMuxer{h}
	for {
		fmt.Printf("Executing Pixie script %s\n", name)
		resultSet, err := vz.ExecuteScript(ctx, script, rm)
		if err != nil && err != io.EOF {
			fmt.Printf("Error while executing Pixie script: %v", err)
		}
		fmt.Printf("Streaming results for %s\n", name)
		if err := resultSet.Stream(); err != nil {
			fmt.Printf("Pixie Streaming error: %v", err)
		}
		resultSet.Close()
		records := h.send(exporter)
		fmt.Printf("Done streaming %v results for %s\n", records, name)
		time.Sleep(10 * time.Second)
	}
}
