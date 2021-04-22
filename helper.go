package main

import (
	"fmt"
	"px.dev/pxapi/types"
	"strings"
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
