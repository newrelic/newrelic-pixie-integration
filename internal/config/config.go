package config

import (
	"github.com/newrelic/infrastructure-agent/pkg/log"
	"strings"
)

var endpoints = map[string]string{
	"eu":  "",
	"stg": "staging.otlp.nr-data.net:4317",
	"fed": "",
	"usa": "",
}

type Config struct {
	Verbose            bool   `default:"false" help:"Print more information to logs."`
	ShowVersion        bool   `default:"false" help:"Print build information and exit"`
	NewRelicRegion     string `help:"Optional. New Relic end point region. Supported values are 'eu','usa','stg' and 'fed'"`
	NewRelicLicenseKey string `help:"License key for the New Relic account"`
	ClusterName        bool   `help:"Name of the monitored cluster"`
	PixieClusterID     bool   `help:"Cluster Id for Pixie"`
	PixieAPIKey        bool   `help:"API Key for Pixie"`
	NewRelicEndpoint   string
}

func (c *Config) Endpoint() (string,error) {
	if c.NewRelicRegion != "" {
		region:=strings.ToLower(c.NewRelicRegion)
		endpoint,ok:=endpoints[region]
		if !ok{
			log.Errorf("Unknown region. Supported values are 'eu', 'usa', 'stg' and 'fed")

		}


		c.NewRelicEndpoint = endpoint
	}
	return "" ,nil
}
