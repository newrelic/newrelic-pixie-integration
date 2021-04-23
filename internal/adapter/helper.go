package adapter

import (
	"fmt"
	"strings"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"px.dev/pxapi/types"
)

const (
	colNamespace = "namespace"
	colService   = "service"
	colPod       = "pod"
	colContainer = "container"
)

func takeNamespaceServiceAndPod(r *types.Record) (ns string, srv string, pod string) {
	ns = r.GetDatum(colNamespace).String()
	nsPrefix := fmt.Sprintf("%s/", ns)
	srv = strings.TrimPrefix(r.GetDatum(colService).String(), nsPrefix)
	pod = strings.TrimPrefix(r.GetDatum(colPod).String(), nsPrefix)
	return
}

func cleanNamespacePrefix(r *types.Record, colNames ...string) []string {
	nsPrefix := fmt.Sprintf("%s/", r.GetDatum(colNamespace))
	out := make([]string, len(colNames))
	for index := range colNames {
		val := r.GetDatum(colNames[index]).String()
		out[index] = strings.TrimPrefix(val, nsPrefix)
	}
	return out
}

func createResource(r *types.Record, cluster string) *resourcepb.Resource {
	namespace, service, pod := takeNamespaceServiceAndPod(r)
	return &resourcepb.Resource{
		Attributes: []*commonpb.KeyValue{
			{
				Key:   "instrumentation.provider",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
			},
			{
				Key:   "service.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: service}},
			},
			{
				Key:   "k8s.namespace.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: namespace}},
			},
			{
				Key:   "service.instance.id",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: pod}},
			},
			{
				Key:   "k8s.pod.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: pod}},
			},
			{
				Key:   "k8s.container.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: r.GetDatum("container").String()}},
			},
			{
				Key:   "k8s.cluster.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: cluster}},
			},
		},
	}
}
