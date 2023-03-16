package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/newrelic/newrelic-pixie-integration/internal/config"
	"github.com/newrelic/newrelic-pixie-integration/internal/pixie"
	"github.com/newrelic/newrelic-pixie-integration/internal/script"
)

const (
	defaultRetries   = 100
	defaultSleepTime = 15 * time.Second
)

func main() {
	ctx := context.Background()

	log.Info("Starting the setup of the New Relic Pixie plugin")
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	clusterId := cfg.Pixie().ClusterID()
	clusterName := cfg.Worker().ClusterName()

	log.Debugf("Setting up Pixie plugin for cluster-id %s", clusterId)
	client, err := setupPixie(ctx, cfg.Pixie(), defaultRetries, defaultSleepTime)
	if err != nil {
		log.WithError(err).Fatal("setting up Pixie client failed")
	}

	log.Debug("Checking the current New Relic plugin configuration")
	plugin, err := client.GetNewRelicPlugin()
	if err != nil {
		log.WithError(err).Fatal("getting data retention plugins failed")
	}

	enablePlugin := true
	if plugin.RetentionEnabled {
		enablePlugin = false
		config, err := client.GetNewRelicPluginConfig()
		if err != nil {
			log.WithError(err).Fatal("getting New Relic plugin config failed")
		}
		if config.ExportUrl != cfg.Exporter().Endpoint() {
			log.Fatal("the New Relic plugin is already installed with a different export URL")
		}
		if config.LicenseKey != cfg.Exporter().LicenseKey() {
			log.Info("New Relic plugin is configured with another license key... Overwriting")
			enablePlugin = true
		}
	}

	if enablePlugin {
		log.Info("Enabling New Relic plugin")
		err := client.EnableNewRelicPlugin(&pixie.NewRelicPluginConfig{
			LicenseKey: cfg.Exporter().LicenseKey(),
			ExportUrl:  cfg.Exporter().Endpoint(),
		}, plugin.LatestVersion)
		if err != nil {
			log.WithError(err).Fatal("failed to enabled New Relic plugin")
		}
	}

	log.Info("Setting up the data retention scripts")

	log.Debug("Getting preset script from the Pixie plugin")
	defsFromPixie, err := client.GetPresetScripts()
	if err != nil {
		log.WithError(err).Fatal("failed to get preset scripts")
	}

	log.Debugf("Getting script definitions from %s", cfg.Worker().ScriptDir())
	defsFromDisk, err := config.ReadScriptDefinitions(cfg.Worker().ScriptDir())
	if err != nil {
		log.WithError(err).Fatalf("failed to read script definitions from %s", cfg.Worker().ScriptDir())
	}

	definitions := append(defsFromPixie, defsFromDisk...)

	log.Debugf("Getting current scripts for cluster")
	currentScripts, err := client.GetClusterScripts(clusterId, clusterName)
	if err != nil {
		log.WithError(err).Fatal("failed to get data retention scripts")
	}

	actions := script.GetActions(definitions, currentScripts, script.ScriptConfig{
		ClusterName:       clusterName,
		ClusterId:         clusterId,
		HttpSpanLimit:     cfg.Worker().HttpSpanLimit(),
		DbSpanLimit:       cfg.Worker().DbSpanLimit(),
		CollectInterval:   cfg.Worker().CollectInterval(),
		ExcludePods:       cfg.Worker().ExcludePods(),
		ExcludeNamespaces: cfg.Worker().ExcludeNamespaces(),
	})

	var errs []error

	for _, s := range actions.ToDelete {
		log.Debugf("Deleting script %s", s.Name)
		err := client.DeleteDataRetentionScript(s.ScriptId)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, s := range actions.ToUpdate {
		log.Debugf("Updating script %s", s.Name)
		err := client.UpdateDataRetentionScript(clusterId, s.ScriptId, s.Name, s.Description, s.FrequencyS, s.Script)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, s := range actions.ToCreate {
		log.Debugf("Creating script %s", s.Name)
		err := client.AddDataRetentionScript(clusterId, s.Name, s.Description, s.FrequencyS, s.Script)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		log.Fatalf("errors while setting up data retention scripts: %v", errs)
	}

	log.Info("All done! The New Relic plugin is now configured.")
	os.Exit(0)
}

func setupPixie(ctx context.Context, cfg config.Pixie, tries int, sleepTime time.Duration) (*pixie.Client, error) {
	for tries > 0 {
		client, err := pixie.NewClient(ctx, cfg.APIKey(), cfg.Host())
		if err == nil {
			return client, nil
		}
		tries -= 1
		log.WithError(err).Warning("error creating Pixie API client")
		time.Sleep(sleepTime)
	}
	return nil, fmt.Errorf("exceeded maximum number of retries")
}
