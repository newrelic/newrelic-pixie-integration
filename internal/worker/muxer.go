package worker

import (
	"context"

	"github.com/newrelic/newrelic-pixie-integration/internal/errors"

	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi"
	"px.dev/pxapi/types"
)

type customHandler interface {
	pxapi.TableRecordHandler
	send(exporter.Exporter) (int64, errors.Error)
}

type ResultMuxer struct {
	RecordHandler pxapi.TableRecordHandler
}

func (r *ResultMuxer) AcceptTable(ctx context.Context, metadata types.TableMetadata) (pxapi.TableRecordHandler, error) {
	return r.RecordHandler, nil
}

type handler struct {
	recordsHandled int64
}

type metricsHandler struct {
	*handler
	adapter adapter.MetricsAdapter
	metrics []*metricpb.ResourceMetrics
}

type spansHandler struct {
	*handler
	adapter adapter.SpansAdapter
	spans   []*tracepb.ResourceSpans
}

func (w *handler) HandleInit(ctx context.Context, metadata types.TableMetadata) error {
	return nil
}

func (w *handler) HandleDone(ctx context.Context) error {
	return nil
}

func (w *metricsHandler) HandleRecord(ctx context.Context, r *types.Record) error {
	w.recordsHandled += 1
	metrics, err := w.adapter.Adapt(r)
	if err != nil {
		return err
	}
	w.metrics = append(w.metrics, metrics)
	return nil
}

func (h *spansHandler) HandleRecord(ctx context.Context, r *types.Record) error {
	h.recordsHandled += 1
	spans, err := h.adapter.Adapt(r)
	if err != nil {
		return err
	}
	h.spans = append(h.spans, spans)
	return nil
}

func (h *spansHandler) send(exporter exporter.Exporter) (int64, errors.Error) {
	if len(h.spans) == 0 {
		return 0, nil
	}
	defer func() {
		h.spans = h.spans[:0]
	}()
	handled := h.recordsHandled
	h.recordsHandled = 0
	if err := exporter.SendSpans(h.spans); err != nil {
		return 0, err
	}
	return handled, nil
}

func (h *metricsHandler) send(exporter exporter.Exporter) (int64, errors.Error) {
	if len(h.metrics) == 0 {
		return 0, nil
	}
	defer func() {
		h.metrics = h.metrics[:0]
	}()
	handled := h.recordsHandled
	h.recordsHandled = 0
	if err := exporter.SendMetrics(h.metrics); err != nil {
		return 0, err
	}
	return handled, nil
}
