package main

import (
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.withpixie.dev/pixie/src/api/go/pxapi/types"
	"time"
)

var DbPXL = `
import px
df = px.DataFrame('mysql_events', start_time='-1m')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']

df = df[['time_', 'service', 'pod', 'namespace', 'req_body', 'latency']]
px.display(df, 'mysql')
`

type dbSpanData struct {
	Timestamp     time.Time
	SpanId        string
	TraceId       string
	Name          string
	Duration      time.Duration
	Service       string
	Pod           string
	ClusterName   string
	Namespace     string
	DbSystem      string
}

func DbSpanHandler(r *types.Record, t *TelemetrySender) error {
	return sendDbSpan(&dbSpanData{
		Timestamp: r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId:  "0",
		TraceId: "0",
		Name: r.GetDatum("req_body").String(),
		Duration: time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Service:  r.GetDatum("service").String(),
		Pod: r.GetDatum("pod").String(),
		ClusterName: t.ClusterName,
		Namespace: r.GetDatum("namespace").String(),
		DbSystem: "mysql",
	}, t)
}

func sendDbSpan(data *dbSpanData, t *TelemetrySender) error {
	return t.Harvester.RecordSpan(telemetry.Span{
		ID:          data.SpanId,
		TraceID:     data.TraceId,
		Name:        data.Name,
		Timestamp:   data.Timestamp,
		Duration:    data.Duration,
		ServiceName: data.Service,
		Attributes: map[string]interface{}{
			"service.instance.id":      data.Pod,
			"span.kind":                "client",
			"instrumentation.provider": "opentelemetry",
			"k8s.cluster.name":         data.ClusterName,
			"k8s.namespace.name":       data.Namespace,
			"k8s.pod.name":             data.Pod,
			"db.system":                data.DbSystem,
		},
	})
}
