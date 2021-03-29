package main

import (
	"context"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"go.withpixie.dev/pixie/src/api/go/pxapi"
	"go.withpixie.dev/pixie/src/api/go/pxapi/types"
	"net/http"
)

const (
	licenseKeyHeader     = "X-License-Key"
	apiKeyHeaderToRemove = "Api-Key"
)

type TelemetrySender struct{
	Harvester      *telemetry.Harvester
	ClusterName    string
	Handler        func(r *types.Record, t *TelemetrySender) error
	RecordsHandled int64
}

func (t *TelemetrySender) HandleInit(ctx context.Context, metadata types.TableMetadata) error {
	return nil
}

func (t *TelemetrySender) HandleRecord(ctx context.Context, r *types.Record) error {
	t.RecordsHandled += 1
	return t.Handler(r, t)
}

func (t *TelemetrySender) HandleDone(ctx context.Context) error {
	return nil
}

func (t *TelemetrySender) GetAndResetRecordsHandled() int64 {
	handled := t.RecordsHandled
	t.RecordsHandled = 0
	return handled
}

type ResultMuxer struct {
	RecordHandler pxapi.TableRecordHandler
}
func (r *ResultMuxer) AcceptTable(ctx context.Context, metadata types.TableMetadata) (pxapi.TableRecordHandler, error) {
	return r.RecordHandler, nil
}

type headerRewriter struct {
	rt http.RoundTripper
}

func newHeaderRewriter(agentTransport http.RoundTripper) *headerRewriter {
	if agentTransport == nil {
		agentTransport = http.DefaultTransport
	}

	return &headerRewriter{
		rt: agentTransport,
	}
}

func (h headerRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	// Use license key header rather than API key
	req.Header.Add(licenseKeyHeader, req.Header.Get(apiKeyHeaderToRemove))
	req.Header.Del(apiKeyHeaderToRemove)
	return h.rt.RoundTrip(req)
}

func WithLicenseHeaderRewriter() func(*telemetry.Config) {
	return func(config *telemetry.Config) {
		config.Client.Transport = newHeaderRewriter(config.Client.Transport)
	}
}
