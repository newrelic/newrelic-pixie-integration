package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/newrelic/infrastructure-agent/pkg/license"
	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/sirupsen/logrus"
)

const (
	envVerbose                   = "VERBOSE"
	envNROTLPHost                = "NR_OTLP_HOST"
	envNRLicenseKEy              = "NR_LICENSE_KEY"
	envPixieClusterID            = "PIXIE_CLUSTER_ID"
	envPixieEndpoint             = "PIXIE_ENDPOINT"
	envPixieAPIKey               = "PIXIE_API_KEY"
	envClusterName               = "CLUSTER_NAME"
	envHttpSpanLimit             = "HTTP_SPAN_LIMIT"
	envDbSpanLimit               = "DB_SPAN_LIMIT"
	envCollectInterval           = "COLLECT_INTERVAL_SEC"
	envHttpMetricCollectInterval = "HTTP_METRIC_COLLECT_INTERVAL_SEC"
	envHttpSpanCollectInterval   = "HTTP_SPAN_COLLECT_INTERVAL_SEC"
	envJvmCollectInterval        = "JVM_COLLECT_INTERVAL_SEC"
	envMysqlCollectInterval      = "MYSQL_COLLECT_INTERVAL_SEC"
	envPostgresCollectInterval   = "POSTGRES_COLLECT_INTERVAL_SEC"
	envExcludePods               = "EXCLUDE_PODS_REGEX"
	envExcludeNamespaces         = "EXCLUDE_NAMESPACES_REGEX"
	defPixieHostname             = "work.withpixie.ai:443"
	endpointEU                   = "otlp.eu01.nr-data.net:4317"
	endpointUSA                  = "otlp.nr-data.net:4317"
	boolTrue                     = "true"
	defHttpSpanLimit             = 1500
	defDbSpanLimit               = 500
	defCollectInterval           = 10
)

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
	log.SetLevel(logrus.InfoLevel)
	if strings.EqualFold(os.Getenv(envVerbose), boolTrue) {
		log.SetLevel(logrus.DebugLevel)
	}
	nrHostname := os.Getenv(envNROTLPHost)
	nrLicenseKey := os.Getenv(envNRLicenseKEy)
	pixieClusterID := os.Getenv(envPixieClusterID)
	pixieAPIKey := os.Getenv(envPixieAPIKey)
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
	httpMetricCollectInterval, err := getIntEnvWithDefault(envHttpMetricCollectInterval, collectInterval)
	if err != nil {
		return err
	}
	httpSpanCollectInterval, err := getIntEnvWithDefault(envHttpSpanCollectInterval, collectInterval)
	if err != nil {
		return err
	}
	jvmCollectInterval, err := getIntEnvWithDefault(envJvmCollectInterval, collectInterval)
	if err != nil {
		return err
	}
	mysqlCollectInterval, err := getIntEnvWithDefault(envMysqlCollectInterval, collectInterval)
	if err != nil {
		return err
	}
	postgresCollectInterval, err := getIntEnvWithDefault(envPostgresCollectInterval, collectInterval)
	if err != nil {
		return err
	}

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
			clusterName:               clusterName,
			pixieClusterID:            pixieClusterID,
			httpSpanLimit:             httpSpanLimit,
			dbSpanLimit:               dbSpanLimit,
			httpMetricCollectInterval: httpMetricCollectInterval,
			httpSpanCollectInterval:   httpSpanCollectInterval,
			jvmCollectInterval:        jvmCollectInterval,
			mysqlCollectInterval:      mysqlCollectInterval,
			postgresCollectInterval:   postgresCollectInterval,
			excludePods:               excludePods,
			excludeNamespaces:         excludeNamespaces,
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
	ClusterName()               string
	PixieClusterID()            string
	HttpSpanLimit()             int64
	DbSpanLimit()               int64
	HttpMetricCollectInterval() int64
	HttpSpanCollectInterval()   int64
	JvmCollectInterval()        int64
	MysqlCollectInterval()      int64
	PostgresCollectInterval()   int64
	ExcludePods()               string
	ExcludeNamespaces()         string
	validate()                  error
}

type worker struct {
	clusterName               string
	pixieClusterID            string
	httpSpanLimit             int64
	dbSpanLimit               int64
	httpMetricCollectInterval int64
	httpSpanCollectInterval   int64
	jvmCollectInterval        int64
	mysqlCollectInterval      int64
	postgresCollectInterval   int64
	excludePods               string
	excludeNamespaces         string
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

func (a *worker) PixieClusterID() string {
	return a.pixieClusterID
}

func (a *worker) HttpSpanLimit() int64 {
	return a.httpSpanLimit
}

func (a *worker) DbSpanLimit() int64 {
	return a.dbSpanLimit
}

func (a *worker) HttpMetricCollectInterval() int64 {
	return a.httpMetricCollectInterval
}

func (a *worker) HttpSpanCollectInterval() int64 {
	return a.httpSpanCollectInterval
}

func (a *worker) JvmCollectInterval() int64 {
	return a.jvmCollectInterval
}

func (a *worker) MysqlCollectInterval() int64 {
	return a.mysqlCollectInterval
}

func (a *worker) PostgresCollectInterval() int64 {
	return a.postgresCollectInterval
}

func (a *worker) ExcludePods() string {
	return a.excludePods
}

func (a *worker) ExcludeNamespaces() string {
	return a.excludeNamespaces
}

func getEndpoint(hostname, licenseKey string) (string, error) {
	if hostname != "" {
		log.Debugf("spans & metrics will be sent to endpoint %s", hostname)
		return hostname, nil
	}
	endpoint := endpointUSA
	nrRegion := license.GetRegion(licenseKey)
	if strings.ToLower(nrRegion) == "eu" {
		endpoint = endpointEU
	}
	log.Debugf("spans & metrics will be sent to endpoint %s", endpoint)
	return endpoint, nil
}
