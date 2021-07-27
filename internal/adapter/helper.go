package adapter

import (
	"fmt"
	"regexp"
	"strings"

	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

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

var regExpIsArray = regexp.MustCompilePOSIX(`\[((\"[a-zA-Z0-9\-\/._]+\")+,)*(\"[a-zA-Z0-9\-\/._]+\")\]`)

func takeNamespaceServiceAndPod(r *types.Record) (ns string, services []string, pod string) {
	ns = r.GetDatum(colNamespace).String()
	nsPrefix := fmt.Sprintf("%s/", ns)
	srv := r.GetDatum(colService).String()
	if regExpIsArray.MatchString(srv) {
		services = strings.Split(srv[1:len(srv)-1], ",")
		for i, name := range services {
			services[i] = strings.TrimPrefix(name[1:len(name)-1], nsPrefix)
		}
	} else {
		services = []string{strings.TrimPrefix(srv, nsPrefix)}
	}
	pod = strings.TrimPrefix(r.GetDatum(colPod).String(), nsPrefix)
	return
}

func createResourceFunc(r *types.Record, namespace, pod, clusterName, clusterId string) func([]string) []resourcepb.Resource {
	resource := resourcepb.Resource{
		Attributes: []*commonpb.KeyValue{
			{
				Key:   "k8s.cluster.id",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: clusterId}},
			},
			{
				Key:   "instrumentation.provider",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: instrumentationName}},
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
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: r.GetDatum(colContainer).String()}},
			},
			{
				Key:   "k8s.cluster.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: clusterName}},
			},
		},
	}
	return func(services []string) []resourcepb.Resource {
		output := make([]resourcepb.Resource, len(services))
		for i, service := range services {
			resource.Attributes = append(resource.Attributes, &commonpb.KeyValue{
				Key:   "service.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: service}},
			})
			output[i] = resource
		}
		return output
	}
}

func createResources(r *types.Record, clusterName, clusterId string) []resourcepb.Resource {
	namespace, services, pod := takeNamespaceServiceAndPod(r)
	return createResourceFunc(r, namespace, pod, clusterName, clusterId)(services)
}

func createArrayOfSpans(resources []resourcepb.Resource, il []*tracepb.InstrumentationLibrarySpans) []*tracepb.ResourceSpans {
	spans := make([]*tracepb.ResourceSpans, len(resources))
	for i := range resources {
		spans[i] = &tracepb.ResourceSpans{
			Resource:                    &resources[i],
			InstrumentationLibrarySpans: il,
		}
	}
	return spans
}

func createArrayOfMetrics(resources []resourcepb.Resource, il []*metricpb.InstrumentationLibraryMetrics) []*metricpb.ResourceMetrics {
	metrics := make([]*metricpb.ResourceMetrics, len(resources))
	for i := range resources {
		metrics[i] = &metricpb.ResourceMetrics{
			Resource:                      &resources[i],
			InstrumentationLibraryMetrics: il,
		}
	}
	return metrics
}
