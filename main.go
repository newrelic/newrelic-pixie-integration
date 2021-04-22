package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi"
	"px.dev/pxapi/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("Setting up OTLP exporter")
	otlpExporter, err := setupOtlpExporter(
		ctx,
		os.Getenv("NR_OTLP_HOST"),
		os.Getenv("NR_LICENSE_KEY"))
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

	go runScript("http", ctx, vz, *otlpExporter, clusterName, spanPXL, SpanHandler)
	go runScript("http_metrics", ctx, vz, *otlpExporter, clusterName, httpMetricsPXL, HttpMetricsHandler)
	go runScript("jvm", ctx, vz, *otlpExporter, clusterName, jvmPXL, JvmHandler)
	go runScript("db_mysql", ctx, vz, *otlpExporter, clusterName, mysqlPXL, MySQLSpanHandler)
	go runScript("db_pg", ctx, vz, *otlpExporter, clusterName, pgsqlPXL, PgSQLSpanHandler)
	select {}
}

func setupPixie(ctx context.Context, apiKey, clusterId string) (*pxapi.VizierClient, error) {
	client, err := pxapi.NewClient(ctx, pxapi.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return client.NewVizierClient(ctx, clusterId)
}

func setupOtlpExporter(ctx context.Context, endpoint string, apiKey string) (*OtlpDataExporter, error) {
	conn, outgoingCtx, err := createConnection(ctx, endpoint, apiKey)
	if err != nil {
		return nil, err
	}
	return &OtlpDataExporter{
		Context:       outgoingCtx,
		MetricsClient: setupMetricsClient(conn),
		TraceClient:   setupTraceClient(conn),
	}, nil
}

func runScript(name string, ctx context.Context, vz *pxapi.VizierClient, e OtlpDataExporter, clusterName string, pxl string, handler func(r *types.Record, t *TelemetrySender) error) {
	t := &TelemetrySender{e, clusterName, handler, 0, make([]*metricpb.ResourceMetrics, 0), make([]*tracepb.ResourceSpans, 0)}
	rm := &ResultMuxer{t}

	for {
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
		time.Sleep(10 * time.Second)
	}
}

type TelemetrySender struct {
	Exporter       OtlpDataExporter
	ClusterName    string
	Handler        func(r *types.Record, t *TelemetrySender) error
	RecordsHandled int64
	Metrics        []*metricpb.ResourceMetrics
	Traces         []*tracepb.ResourceSpans
}

func (t *TelemetrySender) HandleInit(ctx context.Context, metadata types.TableMetadata) error {
	return nil
}

func (t *TelemetrySender) HandleRecord(ctx context.Context, r *types.Record) error {
	t.RecordsHandled += 1
	return t.Handler(r, t)
}

func (t *TelemetrySender) HandleDone(ctx context.Context) error {
	return nil
}

func (t *TelemetrySender) GetAndResetRecordsHandled() int64 {
	handled := t.RecordsHandled
	t.RecordsHandled = 0

	if len(t.Metrics) > 0 {
		if err := t.sendMetrics(); err != nil {
			fmt.Printf("Error sending metrics: %v", err)
		}
		t.Metrics = t.Metrics[:0]
	}

	if len(t.Traces) > 0 {
		if err := t.sendSpans(); err != nil {
			fmt.Printf("Error sending traces: %v", err)
		}
		t.Traces = t.Traces[:0]
	}

	return handled
}

type ResultMuxer struct {
	RecordHandler pxapi.TableRecordHandler
}

func (r *ResultMuxer) AcceptTable(ctx context.Context, metadata types.TableMetadata) (pxapi.TableRecordHandler, error) {
	return r.RecordHandler, nil
}
