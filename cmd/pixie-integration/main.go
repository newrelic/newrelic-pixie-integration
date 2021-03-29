package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.withpixie.dev/pixie/src/api/go/pxapi"
	"go.withpixie.dev/pixie/src/api/go/pxapi/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("Setting up New Relic Telemetry SDK")
	h, err := setupHarvester(
		os.Getenv("NR_LICENSE_KEY"),
		os.Getenv("NR_TRACE_API"),
		os.Getenv("NR_METRIC_API"),
		os.Getenv("VERBOSE"))
	if err != nil {
		fmt.Printf("Error configuring the New Relic Telemetry SDK: %v", err)
		os.Exit(1)
	}

	fmt.Println("Setting up Pixie client")
	vz, err := setupPixie(
		ctx,
		os.Getenv("PIXIE_API_KEY"),
		os.Getenv("PIXIE_CLUSTER_ID"))
	if err != nil {
		fmt.Printf("Error configuring the Pixie client: %v", err)
		os.Exit(2)
	}

	clusterName := os.Getenv("CLUSTER_NAME")

	go runScript("http", ctx, vz, h, clusterName, SpanPXL, SpanHandler)
	go runScript("http_metrics", ctx, vz, h, clusterName, HttpMetricsPXL, HttpMetricsHandler)
	go runScript("jvm", ctx, vz, h, clusterName, JvmPXL, JvmHandler)
	go runScript("db", ctx, vz, h, clusterName, DbPXL, DbSpanHandler)
	select{}
}

func setupHarvester(licenseKey, traceApi, metricApi, verbose string) (*telemetry.Harvester, error) {
	options := []func(*telemetry.Config) {
		telemetry.ConfigAPIKey(licenseKey),
		WithLicenseHeaderRewriter(),
	}
	if traceApi != "" {
		options = append(options, telemetry.ConfigSpansURLOverride(traceApi))
	}
	if metricApi != "" {
		options = append(options, func(cfg *telemetry.Config) {
			cfg.MetricsURLOverride = metricApi
		})
	}
	if verbose != "" {
		options = append(options, telemetry.ConfigBasicDebugLogger(os.Stdout))
	}
	return telemetry.NewHarvester(options...)
}

func setupPixie(ctx context.Context, apiKey, clusterId string) (*pxapi.VizierClient, error) {
	client, err := pxapi.NewClient(ctx, pxapi.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return client.NewVizierClient(ctx, clusterId)
}

func runScript(name string, ctx context.Context, vz *pxapi.VizierClient, h *telemetry.Harvester, clusterName string, pxl string, handler func (r *types.Record, t *TelemetrySender) error) {
	t := &TelemetrySender{h, clusterName, handler, 0}
	rm := &ResultMuxer{t}

	for
	{
		fmt.Printf("Executing Pixie script %s\n", name)
		resultSet, err := vz.ExecuteScript(ctx, pxl, rm)
		if err != nil && err != io.EOF {
			fmt.Printf("Error while executing Pixie script: %v", err)
		}

		fmt.Printf("Streaming results for %s\n", name)
		if err := resultSet.Stream(); err != nil {
			fmt.Printf("Pixie Streaming error: %v", err)
		}
		resultSet.Close()

		fmt.Printf("Done streaming %v results for %s\n", t.GetAndResetRecordsHandled(), name)
		time.Sleep(1 * time.Minute)
	}
}
