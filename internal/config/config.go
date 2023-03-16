package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	envVerbose           = "VERBOSE"
	envNROTLPHost        = "NR_OTLP_HOST"
	envNRLicenseKEy      = "NR_LICENSE_KEY"
	envPixieClusterID    = "PIXIE_CLUSTER_ID"
	envPixieEndpoint     = "PIXIE_ENDPOINT"
	envPixieAPIKey       = "PIXIE_API_KEY"
	envScriptDir         = "SCRIPT_DIR"
	envClusterName       = "CLUSTER_NAME"
	envHttpSpanLimit     = "HTTP_SPAN_LIMIT"
	envDbSpanLimit       = "DB_SPAN_LIMIT"
	envCollectInterval   = "COLLECT_INTERVAL_SEC"
	envExcludePods       = "EXCLUDE_PODS_REGEX"
	envExcludeNamespaces = "EXCLUDE_NAMESPACES_REGEX"
	defScriptDir         = "/scripts"
	defPixieHostname     = "work.withpixie.ai:443"
	endpointEU           = "otlp.eu01.nr-data.net:443"
	endpointUSA          = "otlp.nr-data.net:443"
	boolTrue             = "true"
	defHttpSpanLimit     = 1500
	defDbSpanLimit       = 500
	defCollectInterval   = 30
)

var regionLicenseRegex = regexp.MustCompile(`^([a-z]{2,3})`)

var (
	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""
	once               sync.Once
	instance           Config
)

func GetConfig() (Config, error) {
	var err error
	once.Do(func() {
		err = setUpConfig()
	})
	return instance, err
}

func setUpConfig() error {
	log.SetLevel(log.InfoLevel)
	if strings.EqualFold(os.Getenv(envVerbose), boolTrue) {
		log.SetLevel(log.DebugLevel)
	}
	nrHostname := os.Getenv(envNROTLPHost)
	nrLicenseKey := os.Getenv(envNRLicenseKEy)
	pixieClusterID := os.Getenv(envPixieClusterID)
	pixieAPIKey := os.Getenv(envPixieAPIKey)
	scriptDir := getEnvWithDefault(envScriptDir, defScriptDir)
	clusterName := os.Getenv(envClusterName)
	pixieHost := getEnvWithDefault(envPixieEndpoint, defPixieHostname)
	excludePods := os.Getenv(envExcludePods)
	excludeNamespaces := os.Getenv(envExcludeNamespaces)

	var err error
	httpSpanLimit, err := getIntEnvWithDefault(envHttpSpanLimit, defHttpSpanLimit)
	if err != nil {
		return err
	}
	dbSpanLimit, err := getIntEnvWithDefault(envDbSpanLimit, defDbSpanLimit)
	if err != nil {
		return err
	}
	collectInterval, err := getIntEnvWithDefault(envCollectInterval, defCollectInterval)
	if err != nil {
		return err
	}

	nrHostname = getEndpoint(nrHostname, nrLicenseKey)
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
			scriptDir:         scriptDir,
			clusterName:       clusterName,
			pixieClusterID:    pixieClusterID,
			httpSpanLimit:     httpSpanLimit,
			dbSpanLimit:       dbSpanLimit,
			collectInterval:   collectInterval,
			excludePods:       excludePods,
			excludeNamespaces: excludeNamespaces,
		},
		exporter: &exporter{
			licenseKey: nrLicenseKey,
			endpoint:   nrHostname,
			userAgent:  "pixie/" + integrationVersion,
		},
		pixie: &pixie{
			apiKey:    pixieAPIKey,
			clusterID: pixieClusterID,
			host:      pixieHost,
		},
	}
	return instance.validate()
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getIntEnvWithDefault(key string, defaultValue int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Environment variable %s is not an integer.", key)
	}
	return i, nil
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
	UserAgent() string
	validate() error
}

type exporter struct {
	licenseKey string
	endpoint   string
	userAgent  string
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

func (e *exporter) UserAgent() string {
	return e.userAgent
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
	ScriptDir() string
	ClusterName() string
	PixieClusterID() string
	HttpSpanLimit() int64
	DbSpanLimit() int64
	CollectInterval() int64
	ExcludePods() string
	ExcludeNamespaces() string
	validate() error
}

type worker struct {
	scriptDir         string
	clusterName       string
	pixieClusterID    string
	httpSpanLimit     int64
	dbSpanLimit       int64
	collectInterval   int64
	excludePods       string
	excludeNamespaces string
}

func (a *worker) validate() error {
	if a.clusterName == "" {
		return fmt.Errorf("missing required env variable '%s", envClusterName)
	}
	return nil
}

func (a *worker) ScriptDir() string {
	return a.scriptDir
}

func (a *worker) ClusterName() string {
	return a.clusterName
}

func (a *worker) PixieClusterID() string {
	return a.pixieClusterID
}

func (a *worker) HttpSpanLimit() int64 {
	return a.httpSpanLimit
}

func (a *worker) DbSpanLimit() int64 {
	return a.dbSpanLimit
}

func (a *worker) CollectInterval() int64 {
	return a.collectInterval
}

func (a *worker) ExcludePods() string {
	return a.excludePods
}

func (a *worker) ExcludeNamespaces() string {
	return a.excludeNamespaces
}

func getEndpoint(hostname, licenseKey string) string {
	if hostname != "" {
		log.Debugf("New Relic endpoint is set to %s", hostname)
		return hostname
	}
	endpoint := endpointUSA
	nrRegion := getRegion(licenseKey)
	if strings.ToLower(nrRegion) == "eu" {
		endpoint = endpointEU
	}
	log.Debugf("New Relic endpoint is set to %s", endpoint)
	return endpoint
}

func getRegion(licenseKey string) string {
	matches := regionLicenseRegex.FindStringSubmatch(licenseKey)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
