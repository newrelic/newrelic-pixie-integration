package adapter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"px.dev/pxapi/types"
)

const (
	b3TraceIDPadding = "0000000000000000"
)

const spanPXL = `
import px
df = px.DataFrame('http_events', start_time='-10s')
df.container = df.ctx['container_name']
df.pod = df.ctx['pod']
df.service = df.ctx['service']
df.namespace = df.ctx['namespace']
df.parent_service = px.service_id_to_service_name(px.ip_to_service_id(df.remote_addr))
df.parent_pod = px.pod_id_to_pod_name(px.ip_to_pod_id(df.remote_addr))

df.host = px.pluck(df.req_headers, 'Host')
df.user_agent = px.pluck(df.req_headers, 'User-Agent')
df.trace_id = px.pluck(df.req_headers, 'X-B3-TraceId')
df.span_id = px.pluck(df.req_headers, 'X-B3-SpanId')
df.parent_id = px.pluck(df.req_headers, 'X-B3-ParentSpanId')

df = df[['time_', 'container', 'req_method', 'req_path', 'resp_status', 'latency', 'service', 'pod', 'namespace', 'parent_service', 'parent_pod', 'host', 'trace_id', 'span_id', 'parent_id', 'user_agent']]
px.display(df, 'http')
`

type httpSpans struct {
	clusterName string
}

func (a *httpSpans) ID() string {
	return "http_spans"
}

func (a *httpSpans) Script() string {
	return spanPXL
}

func (a *httpSpans) Adapt(r *types.Record) ([]*tracepb.ResourceSpans, error) {
	spanID, err := getSpanID(r, "span_id")
	if err != nil {
		return nil, err
	}
	traceID, err := getTraceID(r, "trace_id")
	if err != nil {
		return nil, err
	}
	parentSpanID, err := getSpanID(r, "parent_id")
	if err != nil {
		return nil, err
	}
	cleanedValues := cleanNamespacePrefix(r, "parent_service", "parent_pod")

	timestamp := r.GetDatum("time_").(*types.Time64NSValue).Value()
	path := r.GetDatum("req_path").String()
	duration := time.Duration(r.GetDatum("latency").(*types.Int64Value).Value())
	host := r.GetDatum("host").String()
	method := r.GetDatum("req_method").String()
	statusCode := r.GetDatum("resp_status").(*types.Int64Value).Value()
	userAgent := r.GetDatum("user_agent").String()
	resources := createResources(r, a.clusterName)
	return createArrayOfSpans(resources, []*tracepb.InstrumentationLibrarySpans{
		{
			InstrumentationLibrary: instrumentationLibrary,
			Spans: []*tracepb.Span{
				{
					TraceId:           traceID[:],
					SpanId:            spanID[:],
					TraceState:        "",
					ParentSpanId:      parentSpanID[:],
					Name:              urlPolish(path),
					Kind:              tracepb.Span_SPAN_KIND_SERVER,
					StartTimeUnixNano: uint64(timestamp.UnixNano()),
					EndTimeUnixNano:   uint64(timestamp.UnixNano() + duration.Nanoseconds()),
					Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_UNSET},
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "parent.service.name",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: cleanedValues[0]}},
						},
						{
							Key:   "parent.k8s.pod.name",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: cleanedValues[1]}},
						},
						{
							Key:   "http.method",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: method}},
						},
						{
							Key:   "http.url",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: host + path}},
						},
						{
							Key:   "http.target",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: path}},
						},
						{
							Key:   "http.host",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: host}},
						},
						{
							Key:   "http.status_code",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: statusCode}},
						},
						{
							Key:   "http.user_agent",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: userAgent}},
						},
					},
				},
			},
		},
	}), nil
}

var re = regexp.MustCompile(`^([[:xdigit:]]|-|:)+$`)

func urlPolish(url string) string {
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if strings.Contains(part, "?") {
			parts[i] = strings.SplitN(part, "?", 2)[0]
		} else if re.Match([]byte(part)) || len(part) > 32 {
			parts[i] = "<id>"
		}
	}
	return strings.Join(parts, "/")
}

func getTraceID(r *types.Record, colName string) (TraceID, error) {
	var (
		traceID TraceID
		err     error
	)
	value := getString(r, colName, "")
	if value == "" {
		return idGenerator.NewTraceID(), nil
	}
	if len(value) == 16 {
		// Pad 64-bit trace IDs.
		value = b3TraceIDPadding + value
	}
	if traceID, err = TraceIDFromHex(value); err != nil {
		return traceID, fmt.Errorf("error getting TraceID %s from Hex: %w", value, err)
	}
	return traceID, nil
}

func getSpanID(r *types.Record, colName string) (SpanID, error) {
	var (
		spanID SpanID
		err    error
	)
	value := getString(r, colName, "")
	if value == "" {
		return idGenerator.NewSpanID(), nil
	}
	if spanID, err = SpanIDFromHex(value); err != nil {
		return spanID, fmt.Errorf("error getting SpanID %s from Hex: %w", value, err)
	}
	return spanID, nil
}

func getString(r *types.Record, colName string, defaultValue string) string {
	value := r.GetDatum(colName).String()
	if value == "" {
		return defaultValue
	}
	return value
}
