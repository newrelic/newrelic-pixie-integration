package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/config"
	"github.com/newrelic/newrelic-pixie-integration/internal/errors"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	"github.com/newrelic/newrelic-pixie-integration/internal/worker"
	"px.dev/pxapi"
)

var metricsWorker = []adapter.MetricsAdapter{
	adapter.HTTPMetrics,
	adapter.JVM,
}

var spansWorker = []adapter.SpansAdapter{}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err)
		os.Exit(err.ExitStatus())
	}
	log.Debug("Setting up OTLP exporter")
	exporter, err := exporter.New(ctx, cfg.Exporter())
	if err != nil {
		log.Error(err)
		os.Exit(err.ExitStatus())
	}
	log.Debugf("Setting up Pixie client with cluster-id %s\n", cfg.Pixie().ClusterID())
	vz, err := setupPixie(ctx, cfg.Pixie())
	if err != nil {
		log.Error(err)
		os.Exit(err.ExitStatus())
	}
	var wg sync.WaitGroup
	runWorkers(ctx, cfg.Worker(), vz, exporter, &wg)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()
	wg.Wait()
}

func runWorkers(ctx context.Context, cfg config.Worker, vz *pxapi.VizierClient, exporter exporter.Exporter, wg *sync.WaitGroup) {
	w := worker.Build(ctx, cfg, vz, exporter)
	go w.Spans(adapter.HTTPSpans, wg)
	go w.Spans(adapter.MySQL, wg)
	go w.Spans(adapter.PgSQL, wg)
	go w.Metrics(adapter.HTTPMetrics, wg)
	go w.Metrics(adapter.JVM, wg)
	wg.Add(5)
}

func setupPixie(ctx context.Context, cfg config.Pixie) (*pxapi.VizierClient, errors.Error) {
	client, err := pxapi.NewClient(ctx, pxapi.WithAPIKey(cfg.APIKey()))
	if err != nil {
		return nil, errors.ConnectionError(err.Error())
	}
	vz, err := client.NewVizierClient(ctx, cfg.ClusterID())
	if err != nil {
		return nil, errors.ConnectionError(err.Error())
	}
	return vz, nil
}
