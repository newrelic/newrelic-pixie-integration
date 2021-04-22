package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"time"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	instrumentationName    = "pixie"
	instrumentationVersion = "1.0.0"
)

type OtlpDataExporter struct {
	Context       context.Context
	MetricsClient colmetricpb.MetricsServiceClient
	TraceClient   coltracepb.TraceServiceClient
}

func (t *TelemetrySender) ExportHttpMetric(metric *HttpMetricData) error {
	resourceMetrics := transformHttpMetricData(metric)
	t.appendMetric(&resourceMetrics)
	return nil
}

func (t *TelemetrySender) ExportMetric(metric *MetricData) error {
	resourceMetrics := transformMetricData(metric)
	t.appendMetric(&resourceMetrics)
	return nil
}

func (t *TelemetrySender) ExportSpan(span *SpanData) error {
	resourceSpans := transformSpanData(span)
	t.appendSpan(&resourceSpans)
	return nil
}

func (t *TelemetrySender) ExportDbSpan(span *DbSpanData) error {
	resourceSpans := transformDbSpanData(span)
	t.appendSpan(&resourceSpans)
	return nil
}

func setupMetricsClient(conn *grpc.ClientConn) colmetricpb.MetricsServiceClient {
	return colmetricpb.NewMetricsServiceClient(conn)
}

func setupTraceClient(conn *grpc.ClientConn) coltracepb.TraceServiceClient {
	return coltracepb.NewTraceServiceClient(conn)
}

func createConnection(ctx context.Context, endpoint string, apiKey string) (*grpc.ClientConn, context.Context, error) {
	outgoingCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("api-key", apiKey))

	var opts []grpc.DialOption
	tlsDialOption := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	opts = append(opts, tlsDialOption)

	conn, err := grpc.DialContext(outgoingCtx, endpoint, opts...)
	if err != nil {
		return nil, context.Background(), err
	}

	return conn, outgoingCtx, nil
}

func (t *TelemetrySender) appendMetric(resourceMetrics *metricpb.ResourceMetrics) {
	t.Metrics = append(t.Metrics, resourceMetrics)
}

func (t *TelemetrySender) sendMetrics() error {
	_, err := t.Exporter.MetricsClient.Export(t.Exporter.Context, &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: t.Metrics,
	})
	return err
}

func (t *TelemetrySender) appendSpan(resourceSpan *tracepb.ResourceSpans) {
	t.Traces = append(t.Traces, resourceSpan)
}

func (t *TelemetrySender) sendSpans() error {
	_, err := t.Exporter.TraceClient.Export(t.Exporter.Context, &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: t.Traces,
	})
	return err
}

func transformHttpMetricData(data *HttpMetricData) metricpb.ResourceMetrics {
	return metricpb.ResourceMetrics{
		Resource: &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "instrumentation.provider",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
				},
				{
					Key:   "service.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Service}},
				},
				{
					Key:   "service.instance.id",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
				{
					Key:   "k8s.cluster.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ClusterName}},
				},
				{
					Key:   "k8s.namespace.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Namespace}},
				},
				{
					Key:   "k8s.container.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Container}},
				},
				{
					Key:   "k8s.pod.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
			},
		},
		InstrumentationLibraryMetrics: []*metricpb.InstrumentationLibraryMetrics{
			{
				InstrumentationLibrary: &commonpb.InstrumentationLibrary{
					Name:    instrumentationName,
					Version: instrumentationVersion,
				},
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
												Value: strconv.Itoa(int(data.StatusCode)),
											},
										},
										StartTimeUnixNano: uint64(data.Timestamp.UnixNano()),
										TimeUnixNano:      uint64(data.Timestamp.UnixNano() + (10 * time.Second).Nanoseconds()),
										Count:             uint64(data.Count),
										Sum:               data.Sum,
										QuantileValues: []*metricpb.DoubleSummaryDataPoint_ValueAtQuantile{
											{
												Quantile: 0.0,
												Value:    data.Min,
											},
											{
												Quantile: 1.0,
												Value:    data.Max,
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
	}
}

func transformMetricData(data *MetricData) metricpb.ResourceMetrics {
	return metricpb.ResourceMetrics{
		Resource: &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "instrumentation.provider",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
				},
				{
					Key:   "service.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Service}},
				},
				{
					Key:   "service.instance.id",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
				{
					Key:   "k8s.container.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Container}},
				},
				{
					Key:   "k8s.cluster.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ClusterName}},
				},
				{
					Key:   "k8s.namespace.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Namespace}},
				},
				{
					Key:   "k8s.pod.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
			},
		},
		InstrumentationLibraryMetrics: []*metricpb.InstrumentationLibraryMetrics{
			{
				InstrumentationLibrary: &commonpb.InstrumentationLibrary{
					Name:    instrumentationName,
					Version: instrumentationVersion,
				},
				Metrics: []*metricpb.Metric{
					{
						Name:        data.MetricDef.metricName,
						Description: data.MetricDef.description,
						Unit:        data.MetricDef.unit,
						Data: &metricpb.Metric_DoubleGauge{
							DoubleGauge: &metricpb.DoubleGauge{
								DataPoints: []*metricpb.DoubleDataPoint{
									{
										TimeUnixNano: uint64(data.Timestamp.UnixNano()),
										Value:        data.Value,
										Labels:       transformAttributes(data.Attributes),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func transformAttributes(attrs map[string]interface{}) []*commonpb.StringKeyValue {
	stringKeyValues := make([]*commonpb.StringKeyValue, 0)
	for k := range attrs {
		stringKeyValues = append(stringKeyValues, &commonpb.StringKeyValue{
			Key:   k,
			Value: fmt.Sprintf("%v", attrs[k]),
		})
	}
	return stringKeyValues
}

func transformSpanData(data *SpanData) tracepb.ResourceSpans {
	return tracepb.ResourceSpans{
		Resource: &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "instrumentation.provider",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
				},
				{
					Key:   "service.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Service}},
				},
				{
					Key:   "service.instance.id",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
				{
					Key:   "k8s.cluster.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ClusterName}},
				},
				{
					Key:   "k8s.namespace.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Namespace}},
				},
				{
					Key:   "k8s.pod.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
				{
					Key:   "k8s.container.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Container}},
				},
			},
			DroppedAttributesCount: 0,
		},
		InstrumentationLibrarySpans: []*tracepb.InstrumentationLibrarySpans{
			{
				InstrumentationLibrary: &commonpb.InstrumentationLibrary{
					Name:    instrumentationName,
					Version: instrumentationVersion,
				},
				Spans: []*tracepb.Span{
					{
						TraceId:           data.TraceId[:],
						SpanId:            data.SpanId[:],
						TraceState:        "",
						ParentSpanId:      data.ParentId[:],
						Name:              urlPolish(data.Name),
						Kind:              tracepb.Span_SPAN_KIND_SERVER,
						StartTimeUnixNano: uint64(data.Timestamp.UnixNano()),
						EndTimeUnixNano:   uint64(data.Timestamp.UnixNano() + data.Duration.Nanoseconds()),
						Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_UNSET},
						Attributes: []*commonpb.KeyValue{
							{
								Key:   "parent.service.name",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ParentService}},
							},
							{
								Key:   "parent.k8s.pod.name",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ParentPod}},
							},
							{
								Key:   "http.method",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Method}},
							},
							{
								Key:   "http.url",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Host + data.Path}},
							},
							{
								Key:   "http.target",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Path}},
							},
							{
								Key:   "http.host",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Host}},
							},
							{
								Key:   "http.status_code",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: data.StatusCode}},
							},
							{
								Key:   "http.user_agent",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.UserAgent}},
							},
						},
					},
				},
			},
		},
	}
}

func transformDbSpanData(data *DbSpanData) tracepb.ResourceSpans {
	return tracepb.ResourceSpans{
		Resource: &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{
				{
					Key:   "instrumentation.provider",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
				},
				{
					Key:   "service.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Service}},
				},
				{
					Key:   "service.instance.id",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
				{
					Key:   "k8s.container.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Container}},
				},
				{
					Key:   "k8s.cluster.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.ClusterName}},
				},
				{
					Key:   "k8s.namespace.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Namespace}},
				},
				{
					Key:   "k8s.pod.name",
					Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.Pod}},
				},
			},
		},
		InstrumentationLibrarySpans: []*tracepb.InstrumentationLibrarySpans{
			{
				InstrumentationLibrary: &commonpb.InstrumentationLibrary{
					Name:    instrumentationName,
					Version: instrumentationVersion,
				},
				Spans: []*tracepb.Span{
					{
						TraceId:           data.TraceId[:],
						SpanId:            data.SpanId[:],
						TraceState:        "",
						ParentSpanId:      nil,
						Name:              data.Name,
						Kind:              tracepb.Span_SPAN_KIND_CLIENT,
						StartTimeUnixNano: uint64(data.Timestamp.UnixNano()),
						EndTimeUnixNano:   uint64(data.Timestamp.UnixNano() + data.Duration.Nanoseconds()),
						Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_UNSET},
						Attributes: []*commonpb.KeyValue{
							{
								Key:   "db.system",
								Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: data.DbSystem}},
							},
						},
					},
				},
			},
		},
	}
}
