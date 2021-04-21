package main

import (
	"time"

	"px.dev/pxapi/types"
)

const httpMetricsPXL = `
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

type HttpMetricData struct {
	Timestamp   time.Time
	Service     string
	Pod         string
	ClusterName string
	Namespace   string
	StatusCode  int64
	Min         float64
	Max         float64
	Sum         float64
	Count       float64
}

func HttpMetricsHandler(r *types.Record, t *TelemetrySender) error {
	namespace, service, pod := takeNamespaceServiceAndPod(r)
	sendHttpMetric(&HttpMetricData{
		Timestamp:   r.GetDatum("timestamp").(*types.Time64NSValue).Value(),
		Service:     service,
		Pod:         pod,
		ClusterName: t.ClusterName,
		Namespace:   namespace,
		StatusCode:  r.GetDatum("status_code").(*types.Int64Value).Value(),
		Min:         float64(r.GetDatum("latency_min").(*types.Int64Value).Value()) / 1000000,
		Max:         float64(r.GetDatum("latency_max").(*types.Int64Value).Value()) / 1000000,
		Sum:         float64(r.GetDatum("latency_sum").(*types.Int64Value).Value()) / 1000000,
		Count:       float64(r.GetDatum("latency_count").(*types.Int64Value).Value()),
	}, t)

	return nil
}

func sendHttpMetric(data *HttpMetricData, t *TelemetrySender) {
	_ = t.ExportHttpMetric(data)
}
