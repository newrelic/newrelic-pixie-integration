package main

import (
	"context"
	"fmt"
	"os"

	"github.com/newrelic/newrelic-pixie-integration/internal/adapter"
	"github.com/newrelic/newrelic-pixie-integration/internal/exporter"
	"github.com/newrelic/newrelic-pixie-integration/internal/worker"
	"px.dev/pxapi"
)

func main() {
	ctx := context.Background()
	fmt.Println("Setting up OTLP exporter")
	exporter, err := exporter.New(
		ctx,
		os.Getenv("NR_OTLP_HOST"),
		os.Getenv("NR_LICENSE_KEY"),
	)
	if err != nil {
		fmt.Printf("Error configuring the OTLP exporter: %v", err)
		os.Exit(1)
	}

	clusterId := os.Getenv("PIXIE_CLUSTER_ID")

	fmt.Printf("Setting up Pixie client with cluster-id %s\n", clusterId)
	vz, err := setupPixie(
		ctx,
		os.Getenv("PIXIE_API_KEY"),
		clusterId)
	if err != nil {
		fmt.Printf("Error configuring the Pixie client: %v", err)
		os.Exit(2)
	}
	clusterName := os.Getenv("CLUSTER_NAME")

	w := worker.Build(ctx, clusterName, vz, exporter)
	go w.Metrics(adapter.HTTPMetrics)
	go w.Metrics(adapter.JVM)
	go w.Spans(adapter.HTTPSpans)
	go w.Spans(adapter.MySQL)
	go w.Spans(adapter.PgSQL)
	select {}
}

func setupPixie(ctx context.Context, apiKey, clusterId string) (*pxapi.VizierClient, error) {
	client, err := pxapi.NewClient(ctx, pxapi.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return client.NewVizierClient(ctx, clusterId)
}
