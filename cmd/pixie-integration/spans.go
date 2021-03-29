package main

import (
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.withpixie.dev/pixie/src/api/go/pxapi/types"
	"regexp"
	"strings"
	"time"
)

const SpanPXL = `
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

var defaultId = "0"

type spanData struct {
	Timestamp     time.Time
	SpanId        string
	TraceId       string
	ParentId      string
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
	return sendSpan(&spanData{
		Timestamp: r.GetDatum("time_").(*types.Time64NSValue).Value(),
		SpanId: getString(r, "span_id", defaultId),
		TraceId: getString(r, "trace_id", defaultId),
		ParentId: r.GetDatum("parent_id").String(),
		Name: r.GetDatum("req_path").String(),
		Duration: time.Duration(r.GetDatum("latency").(*types.Int64Value).Value()),
		Service:  r.GetDatum("service").String(),
		Pod: r.GetDatum("pod").String(),
		ClusterName: t.ClusterName,
		Namespace: r.GetDatum("namespace").String(),
		Host: r.GetDatum("host").String(),
		Method: r.GetDatum("req_method").String(),
		Path: r.GetDatum("req_path").String(),
		StatusCode: r.GetDatum("resp_status").(*types.Int64Value).Value(),
		UserAgent: r.GetDatum("user_agent").String(),
		ParentService: r.GetDatum("parent_service").String(),
		ParentPod: r.GetDatum("parent_pod").String(),
	}, t)
}

func getString(r *types.Record, colName string, defaultValue string) string {
	value := r.GetDatum(colName).String()
	if value == "" {
		return defaultValue
	}
	return value
}

func sendSpan(data *spanData, t *TelemetrySender) error {
	attributes := map[string]interface{}{
		"http.host":                data.Host,
		"http.method":              data.Method,
		"http.path":                data.Path,
		"http.status_code":         data.StatusCode,
		"http.url":                 data.Host + data.Path,
		"http.user_agent":          data.UserAgent,
		"service.instance.id":      data.Pod,
		"span.kind":                "server",
		"instrumentation.provider": "opentelemetry",
		"instrumentation.name":     "pixie",
		"k8s.cluster.name":         data.ClusterName,
		"k8s.namespace.name":       data.Namespace,
		"k8s.pod.name":             data.Pod,
		"parent.service.name":      data.ParentService,
		"parent.k8s.pod.name":      data.ParentPod,
	}

	if data.ParentId != "" {
		attributes["parent.id"] = data.ParentId
	}

	return t.Harvester.RecordSpan(telemetry.Span{
		ID:          data.SpanId,
		TraceID:     data.TraceId,
		Name:        urlPolish(data.Name),
		Timestamp:   data.Timestamp,
		Duration:    data.Duration,
		ServiceName: data.Service,
		Attributes:  attributes,
	})
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
