package adapter

import (
	"strconv"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"px.dev/pxapi/types"
)

const httpMetricsPXL = `
import px

ns_per_ms = 1000 * 1000
ns_per_s = 1000 * ns_per_ms
window_ns = px.DurationNanos(10 * ns_per_s)

df = px.DataFrame(table='http_events', start_time='-10s')

df.timestamp = px.bin(df.time_, window_ns)

df.container = df.ctx['container_name']
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']

df.status_code = df.resp_status

df = df.groupby(['timestamp', 'status_code', 'pod', 'container','service', 'namespace']).agg(
    latency_min=('latency', px.min),
    latency_max=('latency', px.max),
    latency_sum=('latency', px.sum),
    latency_count=('latency', px.count)
)

px.display(df, 'http')
`

type httpMetrics struct {
	clusterName string
}

func (a *httpMetrics) ID() string {
	return "http_metrics"
}

func (a *httpMetrics) Script() string {
	return httpMetricsPXL
}

func (a *httpMetrics) Adapt(r *types.Record) (*metricpb.ResourceMetrics, error) {
	timestamp := r.GetDatum("timestamp").(*types.Time64NSValue).Value()
	statusCode := r.GetDatum("status_code").(*types.Int64Value).Value()
	latMin := float64(r.GetDatum("latency_min").(*types.Int64Value).Value()) / 1000000
	latMax := float64(r.GetDatum("latency_max").(*types.Int64Value).Value()) / 1000000
	latSum := float64(r.GetDatum("latency_sum").(*types.Int64Value).Value()) / 1000000
	latCount := float64(r.GetDatum("latency_count").(*types.Int64Value).Value())
	return &metricpb.ResourceMetrics{
		Resource: createResource(r, a.clusterName),
		InstrumentationLibraryMetrics: []*metricpb.InstrumentationLibraryMetrics{
			{
				InstrumentationLibrary: instrumentationLibrary,
				Metrics: []*metricpb.Metric{
					{
						Name:        "http.server.duration",
						Description: "measures the duration of the inbound HTTP request",
						Unit:        "ms",
						Data: &metricpb.Metric_DoubleSummary{
							DoubleSummary: &metricpb.DoubleSummary{
								DataPoints: []*metricpb.DoubleSummaryDataPoint{
									{
										Labels: []*commonpb.StringKeyValue{
											{
												Key:   "http.status_code",
												Value: strconv.Itoa(int(statusCode)),
											},
										},
										StartTimeUnixNano: uint64(timestamp.UnixNano()),
										TimeUnixNano:      uint64(timestamp.UnixNano() + (10 * time.Second).Nanoseconds()),
										Count:             uint64(latCount),
										Sum:               latSum,
										QuantileValues: []*metricpb.DoubleSummaryDataPoint_ValueAtQuantile{
											{
												Quantile: 0.0,
												Value:    latMin,
											},
											{
												Quantile: 1.0,
												Value:    latMax,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
