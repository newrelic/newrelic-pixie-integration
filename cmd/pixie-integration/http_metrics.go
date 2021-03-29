package main

import (
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.withpixie.dev/pixie/src/api/go/pxapi/types"
	"time"
)

const HttpMetricsPXL = `
import px

ns_per_ms = 1000 * 1000
ns_per_s = 1000 * ns_per_ms
window_ns = px.DurationNanos(10 * ns_per_s)

df = px.DataFrame(table='http_events', start_time='-1m')

df.timestamp = px.bin(df.time_, window_ns)

df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']

df.status_code = df.resp_status

df = df.groupby(['timestamp', 'status_code', 'pod', 'service', 'namespace']).agg(
    latency_min=('latency', px.min),
    latency_max=('latency', px.max),
    latency_sum=('latency', px.sum),
    latency_count=('latency', px.count)
)

px.display(df, 'http')
`

type httpMetricData struct {
	Timestamp     time.Time
	Service       string
	Pod           string
	ClusterName   string
	Namespace     string
	StatusCode    int64
	Min           float64
	Max           float64
	Sum           float64
	Count         float64
}

func HttpMetricsHandler(r *types.Record, t *TelemetrySender) error {
	sendHttpMetric(&httpMetricData{
		Timestamp: r.GetDatum("timestamp").(*types.Time64NSValue).Value(),
		Service:  r.GetDatum("service").String(),
		Pod: r.GetDatum("pod").String(),
		ClusterName: t.ClusterName,
		Namespace: r.GetDatum("namespace").String(),
		StatusCode: r.GetDatum("status_code").(*types.Int64Value).Value(),
		Min: float64(r.GetDatum("latency_min").(*types.Int64Value).Value()) / 1000000,
		Max: float64(r.GetDatum("latency_max").(*types.Int64Value).Value()) / 1000000,
		Sum: float64(r.GetDatum("latency_sum").(*types.Int64Value).Value()) / 1000000,
		Count: float64(r.GetDatum("latency_count").(*types.Int64Value).Value()),
	}, t)

	return nil
}

func sendHttpMetric(data *httpMetricData, t *TelemetrySender) {
	attributes := map[string]interface{}{
		"service.name" :            data.Service,
		"service.instance.id":      data.Pod,
		"instrumentation.provider": "opentelemetry",
		"instrumentation.name":     "pixie",
		"k8s.cluster.name":         data.ClusterName,
		"k8s.namespace.name":       data.Namespace,
		"k8s.pod.name":             data.Pod,
		"http.status_code":         data.StatusCode,
	}

	t.Harvester.RecordMetric(telemetry.Summary{
		Timestamp: data.Timestamp,
		Name: "http.server.duration",
		Count: data.Count,
		Sum: data.Sum,
		Min: data.Min,
		Max: data.Max,
		Interval: 10 * time.Second,
		Attributes: attributes,
	})
}
