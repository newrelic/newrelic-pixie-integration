package worker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/config"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi"
)

const (
	defaultSleepTime = 10 * time.Second
	maxExecutionTime = 9 * time.Second
)

type Worker interface {
	Spans(adapter.SpansAdapter, *sync.WaitGroup)
	Metrics(adapter.MetricsAdapter, *sync.WaitGroup)
}

type worker struct {
	ctx         context.Context
	clusterName string
	vz          *pxapi.VizierClient
	exporter    exporter.Exporter
}

func Build(ctx context.Context, cfg config.Worker, vz *pxapi.VizierClient, exporter exporter.Exporter) Worker {
	return &worker{
		ctx:         ctx,
		clusterName: cfg.ClusterName(),
		vz:          vz,
		exporter:    exporter,
	}
}

func (w *worker) Metrics(adapter adapter.MetricsAdapter, wg *sync.WaitGroup) {
	h := &metricsHandler{
		handler: &handler{},
		adapter: adapter,
		metrics: make([]*metricpb.ResourceMetrics, 0),
	}
	run(w.ctx, wg, adapter.ID(), adapter.Script(), w.vz, h, w.exporter)
}

func (w *worker) Spans(adapter adapter.SpansAdapter, wg *sync.WaitGroup) {
	h := &spansHandler{
		handler: &handler{},
		adapter: adapter,
		spans:   make([]*tracepb.ResourceSpans, 0),
	}
	run(w.ctx, wg, adapter.ID(), adapter.Script(), w.vz, h, w.exporter)
}

func run(ctx context.Context, wg *sync.WaitGroup, name string, script string, vz *pxapi.VizierClient, h customHandler, exporter exporter.Exporter) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn(err)
			log.Infof("sleep 10 seconds to be recovered")
			time.Sleep(defaultSleepTime)
			run(ctx, wg, name, script, vz, h, exporter)
		}
	}()
	rm := &ResultMuxer{h}
	for {
		var resultSet *pxapi.ScriptResults
		select {
		case <-ctx.Done():
			log.Infof("leaving worker for %s", name)
			wg.Done()
			return
		default:
			start := time.Now()
			ch := make(chan error, 1)
			pixieCtx, cancelFn := context.WithCancel(ctx)
			go func() {
				log.Debugf("executing Pixie script %s\n", name)
				resultSet, err := vz.ExecuteScript(pixieCtx, script, rm)
				if err != nil && err != io.EOF {
					ch <- fmt.Errorf("error while executing Pixie script: %s", err)
					return
				}
				log.Debugf("streaming results for %s\n", name)
				if err := resultSet.Stream(); err != nil {
					ch <- fmt.Errorf("pixie streaming error: %s", err)
					return
				}
				records := h.send(exporter)
				log.Debugf("done streaming %d results for %s\n", records, name)
				ch <- nil
			}()
			select {
			case err := <-ch:
				if err == nil {
					log.Debugf("execution completed successfully for %s!", name)
				} else {
					log.Warnf("execution failed for %s: %s", name, err)
				}
			case <-time.After(maxExecutionTime):
				cancelFn()
				log.Warnf("execution out of time for %s!", name)
			}
			if resultSet != nil {
				resultSet.Close()
			}
			sleepTime := start.Add(defaultSleepTime).Sub(time.Now())
			if (sleepTime > 0) {
				time.Sleep(sleepTime)
			} else {
				log.Warnf("skipping the sleep for %s!", name)
			}
		}
	}
}
