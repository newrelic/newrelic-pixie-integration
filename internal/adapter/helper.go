package adapter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/newrelic/infrastructure-agent/pkg/log"

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

type ResourceHelper struct {
	excludePods       *regexp.Regexp
	excludeNamespaces *regexp.Regexp
}

func NewResourceHelper(excludePods , excludeNamespaces string) (*ResourceHelper, error) {
	var rExcludePods *regexp.Regexp
	if excludePods != "" {
		log.Infof("Excluding pods matching regex '%s'", excludePods)
		var err error
		rExcludePods, err = regexp.Compile(excludePods)
		if err != nil {
			return nil, err
		}
	}

	var rExcludeNamespaces *regexp.Regexp
	if excludeNamespaces != "" {
		log.Infof("Excluding namespaces matching regex '%s'", excludeNamespaces)
		var err error
		rExcludeNamespaces, err = regexp.Compile(excludeNamespaces)
		if err != nil {
			return nil, err
		}
	}

	return &ResourceHelper{
		rExcludePods,
		rExcludeNamespaces,
	}, nil
}

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

func createResourceFunc(r *types.Record, namespace, pod, clusterName, pixieClusterID string) func([]string) []resourcepb.Resource {
	resource := resourcepb.Resource{
		Attributes: []*commonpb.KeyValue{
			{
				Key:   "pixie.cluster.id",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: pixieClusterID}},
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

func (rh *ResourceHelper) createResources(r *types.Record, clusterName, pixieClusterID string) []resourcepb.Resource {
	namespace, services, pod := takeNamespaceServiceAndPod(r)
	if rh.shouldFilter(namespace, pod) {
		return []resourcepb.Resource{}
	}
	return createResourceFunc(r, namespace, pod, clusterName, pixieClusterID)(services)
}

func (rh *ResourceHelper) shouldFilter(namespace, pod string) bool {
	if rh.excludeNamespaces != nil && rh.excludeNamespaces.MatchString(namespace) {
		return true
	}
	if rh.excludePods != nil && rh.excludePods.MatchString(pod) {
		return true
	}
	return false
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
