package adapter

import (
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi/types"
)

var pgsqllPXL = `
import px
df = px.DataFrame('pgsql_events', start_time='-10s')
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']	
df.container = df.ctx['container_name']

df = df[['time_', 'container', 'service', 'pod', 'namespace', 'req', 'latency']]
px.display(df, 'pgsql')
`

type pogsql struct {
	clusterName string
}

func (a *pogsql) ID() string {
	return "db_postgres"
}

func (a *pogsql) Script() string {
	return pgsqllPXL
}

func (a *pogsql) Adapt(r *types.Record) (*tracepb.ResourceSpans, error) {
	spanID := idGenerator.NewSpanID()
	traceID := idGenerator.NewTraceID()
	timestamp := r.GetDatum("time_").(*types.Time64NSValue).Value()
	duration := time.Duration(r.GetDatum("latency").(*types.Int64Value).Value())
	return &tracepb.ResourceSpans{
		Resource: createResource(r, a.clusterName),
		InstrumentationLibrarySpans: []*tracepb.InstrumentationLibrarySpans{
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
						Name:              r.GetDatum("req").String(),
						Attributes: []*commonpb.KeyValue{
							{
								Key:   "db.system",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "postgres"}},
							},
						},
					},
				},
			},
		},
	}, nil
}
