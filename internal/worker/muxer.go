package worker

import (
	"context"

	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi"
	"px.dev/pxapi/types"
)

type customHandler interface {
	pxapi.TableRecordHandler
	send(exporter.Exporter) int
}

type ResultMuxer struct {
	RecordHandler pxapi.TableRecordHandler
}

func (r *ResultMuxer) AcceptTable(ctx context.Context, metadata types.TableMetadata) (pxapi.TableRecordHandler, error) {
	return r.RecordHandler, nil
}

type handler struct{}

type metricsHandler struct {
	*handler
	adapter        adapter.MetricsAdapter
	resourceHelper *adapter.ResourceHelper
	metrics        []*metricpb.ResourceMetrics
}

type spansHandler struct {
	*handler
	adapter        adapter.SpansAdapter
	resourceHelper *adapter.ResourceHelper
	spans          []*tracepb.ResourceSpans
}

func (w *handler) HandleInit(ctx context.Context, metadata types.TableMetadata) error {
	return nil
}

func (w *handler) HandleDone(ctx context.Context) error {
	return nil
}

func (h *metricsHandler) HandleRecord(ctx context.Context, r *types.Record) error {
	metrics, err := h.adapter.Adapt(h.resourceHelper, r)
	if err != nil {
		return err
	}
	h.metrics = append(h.metrics, metrics...)
	return nil
}

func (h *spansHandler) HandleRecord(ctx context.Context, r *types.Record) error {
	spans, err := h.adapter.Adapt(h.resourceHelper, r)
	if err != nil {
		return err
	}
	h.spans = append(h.spans, spans...)
	return nil
}

func (h *spansHandler) send(exporter exporter.Exporter) int {
	defer func() {
		h.spans = h.spans[:0]
	}()
	return exporter.SendSpans(h.spans)
}

func (h *metricsHandler) send(exporter exporter.Exporter) int {
	defer func() {
		h.metrics = h.metrics[:0]
	}()
	return exporter.SendMetrics(h.metrics)
}
