package main

import (
	"px.dev/pxapi/types"
	"time"
)

var dbPXL = `
import px
df = px.DataFrame('mysql_events', start_time='-1m')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']

df = df[['time_', 'service', 'pod', 'namespace', 'req_body', 'latency']]
px.display(df, 'mysql')
`

type DbSpanData struct {
	Timestamp   time.Time
	SpanId      SpanID
	TraceId     TraceID
	Name        string
	Duration    time.Duration
	Service     string
	Pod         string
	ClusterName string
	Namespace   string
	DbSystem    string
}

func DbSpanHandler(r *types.Record, t *TelemetrySender) error {
	namespace, service, pod := takeNamespaceServiceAndPod(r)
	return sendDbSpan(&DbSpanData{
		Timestamp:   r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId:      idGenerator.NewSpanID(),
		TraceId:     idGenerator.NewTraceID(),
		Name:        r.GetDatum("req_body").String(),
		Duration:    time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Service:     service,
		Pod:         pod,
		ClusterName: t.ClusterName,
		Namespace:   namespace,
		DbSystem:    "mysql",
	}, t)
}

func sendDbSpan(data *DbSpanData, t *TelemetrySender) error {
	return t.ExportDbSpan(data)
}
