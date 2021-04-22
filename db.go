package main

import (
	"px.dev/pxapi/types"
	"time"
)

var mysqlPXL = `
import px
df = px.DataFrame('mysql_events', start_time='-10s')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']	
df.container = df.ctx['container_name']

df = df[['time_', 'container', 'service', 'pod', 'namespace', 'req_body', 'latency']]
px.display(df, 'mysql')
`

var pgsqlPXL = `
import px
df = px.DataFrame('pgsql_events', start_time='-10s')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']
df.container = df.ctx['container_name']

df = df[['time_', 'container', 'service', 'pod', 'namespace', 'req', 'latency']]
px.display(df, 'pgsql')
`

type DbSpanData struct {
	Timestamp   time.Time
	SpanId      SpanID
	TraceId     TraceID
	Name        string
	Duration    time.Duration
	Container   string
	Service     string
	Pod         string
	ClusterName string
	Namespace   string
	DbSystem    string
}

func MySQLSpanHandler(r *types.Record, t *TelemetrySender) error {
	namespace, service, pod := takeNamespaceServiceAndPod(r)
	return sendDbSpan(&DbSpanData{
		Timestamp:   r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId:      idGenerator.NewSpanID(),
		TraceId:     idGenerator.NewTraceID(),
		Name:        r.GetDatum("req_body").String(),
		Duration:    time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Container:   r.GetDatum("container").String(),
		Service:     service,
		Pod:         pod,
		ClusterName: t.ClusterName,
		Namespace:   namespace,
		DbSystem:    "mysql",
	}, t)
}

func PgSQLSpanHandler(r *types.Record, t *TelemetrySender) error {
	namespace, service, pod := takeNamespaceServiceAndPod(r)
	return sendDbSpan(&DbSpanData{
		Timestamp:   r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId:      idGenerator.NewSpanID(),
		TraceId:     idGenerator.NewTraceID(),
		Name:        r.GetDatum("req").String(),
		Duration:    time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Container:   r.GetDatum("container").String(),
		Service:     service,
		Pod:         pod,
		ClusterName: t.ClusterName,
		Namespace:   namespace,
		DbSystem:    "postgres",
	}, t)
}



func sendDbSpan(data *DbSpanData, t *TelemetrySender) error {
	return t.ExportDbSpan(data)
}
