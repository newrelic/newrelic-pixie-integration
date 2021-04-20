package main

import (
	"regexp"
	"strings"
	"time"

	"px.dev/pxapi/types"
)

const spanPXL = `
import px
df = px.DataFrame('http_events', start_time='-1m')
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

df = df[['time_', 'req_method', 'req_path', 'resp_status', 'latency', 'service', 'pod', 'namespace', 'parent_service', 'parent_pod', 'host', 'trace_id', 'span_id', 'parent_id', 'user_agent']]
px.display(df, 'http')
`

type SpanData struct {
	Timestamp     time.Time
	SpanId        SpanID
	TraceId       TraceID
	ParentId      SpanID
	Name          string
	Duration      time.Duration
	Service       string
	Pod           string
	ClusterName   string
	Namespace     string
	Host          string
	Method        string
	Path          string
	StatusCode    int64
	UserAgent     string
	ParentService string
	ParentPod     string
}

func SpanHandler(r *types.Record, t *TelemetrySender) error {
	spanID, err := getSpanID(r, "span_id")
	if err != nil {
		return err
	}
	traceID, err := getTraceID(r, "trace_id")
	if err != nil {
		return err
	}
	parentSpanID, err := getSpanID(r, "parent_id")
	if err != nil {
		return err
	}

	return sendSpan(&SpanData{
		Timestamp:     r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId:        spanID,
		TraceId:       traceID,
		ParentId:      parentSpanID,
		Name:          r.GetDatum("req_path").String(),
		Duration:      time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Service:       r.GetDatum("service").String(),
		Pod:           r.GetDatum("pod").String(),
		ClusterName:   t.ClusterName,
		Namespace:     r.GetDatum("namespace").String(),
		Host:          r.GetDatum("host").String(),
		Method:        r.GetDatum("req_method").String(),
		Path:          r.GetDatum("req_path").String(),
		StatusCode:    r.GetDatum("resp_status").(*types.Int64Value).Value(),
		UserAgent:     r.GetDatum("user_agent").String(),
		ParentService: r.GetDatum("parent_service").String(),
		ParentPod:     r.GetDatum("parent_pod").String(),
	}, t)
}

const (
	b3TraceIDPadding = "0000000000000000"
)

var (
	idGenerator = defaultIDGenerator()
)

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
		return traceID, err
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
		return spanID, err
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

func sendSpan(data *SpanData, t *TelemetrySender) error {
	return t.ExportSpan(data)
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
