package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/newrelic/infrastructure-agent/pkg/license"
	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/sirupsen/logrus"
)

const (
	envVerbose        = "VERBOSE"
	envNROTLPHost     = "NR_OTLP_HOST"
	envNRLicenseKEy   = "NR_LICENSE_KEY"
	envPixieClusterID = "PIXIE_CLUSTER_ID"
	envPixieHost      = "PIXIE_HOST"
	envPixieAPIKey    = "PIXIE_API_KEY"
	envClusterName    = "CLUSTER_NAME"
	defPixieHostname  = "work.withpixie.ai:443"
	endpointEU        = "otlp.eu01.nr-data.net:4317"
	endpointUSA       = "otlp.nr-data.net:4317"
	endpointStg       = "staging.otlp.nr-data.net:4317"
	boolTrue          = "true"
)

var (
	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""
	endpoints          = map[string]string{
		"eu": endpointEU,
		"":   endpointUSA,
	}
	once     sync.Once
	instance Config
)

func GetConfig() (Config, error) {
	var err error
	once.Do(func() {
		err = setUpConfig()
	})
	return instance, err
}

func setUpConfig() error {
	log.SetLevel(logrus.InfoLevel)
	if strings.EqualFold(os.Getenv(envVerbose), boolTrue) {
		log.SetLevel(logrus.DebugLevel)
	}
	nrHostname := os.Getenv(envNROTLPHost)
	nrLicenseKey := os.Getenv(envNRLicenseKEy)
	pixieClusterID := os.Getenv(envPixieClusterID)
	pixieAPIKey := os.Getenv(envPixieAPIKey)
	clusterName := os.Getenv(envClusterName)
	pixieHost := os.Getenv(envPixieHost)
	if pixieHost == "" {
		pixieHost = defPixieHostname
	}
	var err error
	nrHostname, err = getEndpoint(nrHostname, nrLicenseKey)
	if err != nil {
		return fmt.Errorf("error getting endpoint for license: %w", err)
	}
	instance = &config{
		settings: &settings{
			buildDate: buildDate,
			commit:    gitCommit,
			version:   integrationVersion,
		},
		worker: &worker{
			clusterName: clusterName,
		},
		exporter: &exporter{
			licenseKey: nrLicenseKey,
			endpoint:   nrHostname,
		},
		pixie: &pixie{
			apiKey:    pixieAPIKey,
			clusterID: pixieClusterID,
			host:      pixieHost,
		},
	}
	return instance.validate()
}

type Config interface {
	Verbose() bool
	Settings() Settings
	Exporter() Exporter
	Pixie() Pixie
	Worker() Worker
	validate() error
}

type config struct {
	verbose  bool
	worker   Worker
	exporter Exporter
	pixie    Pixie
	settings Settings
}

func (c *config) validate() error {
	if err := c.Pixie().validate(); err != nil {
		return fmt.Errorf("error validating pixie config: %w", err)
	}
	if err := c.Worker().validate(); err != nil {
		return fmt.Errorf("error validating worker config: %w", err)
	}
	return c.Exporter().validate()
}

func (c *config) Settings() Settings {
	return c.settings
}

func (c *config) Verbose() bool {
	return c.verbose
}

func (c *config) Exporter() Exporter {
	return c.exporter
}

func (c *config) Worker() Worker {
	return c.worker
}

func (c *config) Pixie() Pixie {
	return c.pixie
}

type Settings interface {
	Version() string
	Commit() string
	BuildDate() string
}

type settings struct {
	buildDate string
	commit    string
	version   string
}

func (s *settings) Version() string {
	return s.version
}

func (s *settings) Commit() string {
	return s.commit
}

func (s *settings) BuildDate() string {
	return s.buildDate
}

type Exporter interface {
	LicenseKey() string
	Endpoint() string
	validate() error
}

type exporter struct {
	licenseKey string
	endpoint   string
}

func (e *exporter) validate() error {
	if e.licenseKey == "" {
		return fmt.Errorf("missing required env variable '%s", envNRLicenseKEy)
	}
	return nil
}

func (e *exporter) LicenseKey() string {
	return e.licenseKey
}

func (e *exporter) Endpoint() string {
	return e.endpoint
}

type Pixie interface {
	APIKey() string
	ClusterID() string
	Host() string
	validate() error
}

type pixie struct {
	apiKey    string
	clusterID string
	host      string
}

func (p *pixie) validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("missing required env variable '%s", envPixieAPIKey)
	}
	if p.clusterID == "" {
		return fmt.Errorf("missing required env variable '%s", envPixieClusterID)
	}
	return nil
}

func (p *pixie) APIKey() string {
	return p.apiKey
}

func (p *pixie) ClusterID() string {
	return p.clusterID
}

func (p *pixie) Host() string {
	return p.host
}

type Worker interface {
	ClusterName() string
	validate() error
}

type worker struct {
	clusterName string
}

func (a *worker) validate() error {
	if a.clusterName == "" {
		return fmt.Errorf("missing required env variable '%s", envClusterName)
	}
	return nil
}

func (a *worker) ClusterName() string {
	return a.clusterName
}

func getEndpoint(hostname, licenseKey string) (string, error) {
	if hostname != "" {
		log.Debugf("spans & metrics will be %s", hostname)
		return hostname, nil
	}
	nrRegion := license.GetRegion(licenseKey)
	endpoint, ok := endpoints[strings.ToLower(nrRegion)]
	if !ok {
		return "", fmt.Errorf("the provided license key doesn't belong to a known New Relic region")
	}
	log.Debugf("spans & metrics will be sent to endpoint %s", endpoint)
	return endpoint, nil
}
