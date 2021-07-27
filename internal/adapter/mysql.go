package adapter

import (
	"fmt"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi/types"
)

const mysqlTemplate = `
#px:set max_output_rows_per_table=%d

import px
df = px.DataFrame('mysql_events', start_time='-%ds')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']	
df.container = df.ctx['container_name']

df.normed_query_struct = px.normalize_mysql(df.req_body, df.req_cmd)
df.query = px.pluck(df.normed_query_struct, 'query')
df = df[df.query != ""]

df = df[['time_', 'container', 'service', 'pod', 'namespace', 'query', 'latency']]
px.display(df, 'mysql')
`

type mysql struct {
	clusterName        string
	clusterID          string
	collectIntervalSec int64
	script             string
}

func newMysql(clusterName, clusterID string, collectIntervalSec, spanLimit int64) *mysql {
	return &mysql{clusterName, clusterID, collectIntervalSec, fmt.Sprintf(mysqlTemplate, spanLimit, collectIntervalSec)}
}

func (a *mysql) ID() string {
	return "db_mysql"
}

func (a *mysql) CollectIntervalSec() int64 {
	return a.collectIntervalSec
}

func (a *mysql) Script() string {
	return a.script
}

func (a *mysql) Adapt(r *types.Record) ([]*tracepb.ResourceSpans, error) {
	spanID := idGenerator.NewSpanID()
	traceID := idGenerator.NewTraceID()
	timestamp := r.GetDatum("time_").(*types.Time64NSValue).Value()
	duration := time.Duration(r.GetDatum("latency").(*types.Int64Value).Value())
	resources := createResources(r, a.clusterName, a.clusterID)
	return createArrayOfSpans(resources, []*tracepb.InstrumentationLibrarySpans{
		{
			InstrumentationLibrary: instrumentationLibrary,
			Spans: []*tracepb.Span{
				{
					TraceId:           traceID[:],
					SpanId:            spanID[:],
					TraceState:        "",
					ParentSpanId:      nil,
					Kind:              tracepb.Span_SPAN_KIND_CLIENT,
					StartTimeUnixNano: uint64(timestamp.UnixNano()),
					EndTimeUnixNano:   uint64(timestamp.UnixNano() + duration.Nanoseconds()),
					Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_UNSET},
					Name:              r.GetDatum("query").String(),
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "db.system",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "mysql"}},
						},
					},
				},
			},
		},
	}), nil
}
