package script

import (
	"fmt"
	"strings"
)

const (
	scriptPrefix          = "nri-"
	httpMetricsScript     = "HTTP Metrics"
	httpSpansScript       = "HTTP Spans"
	jvmMetricsScript      = "JVM Metrics"
	mysqlSpansScript      = "MySQL Spans"
	postgresqlSpansScript = "PostgreSQL Spans"
)

type ScriptConfig struct {
	ClusterName               string
	ClusterId                 string
	HttpSpanLimit             int64
	DbSpanLimit               int64
	CollectInterval           int64
	HttpMetricCollectInterval int64
	HttpSpanCollectInterval   int64
	JvmCollectInterval        int64
	MysqlCollectInterval      int64
	PostgresCollectInterval   int64
	ExcludePods               string
	ExcludeNamespaces         string
}

type Script struct {
	ScriptDefinition
	ScriptId   string
	ClusterIds string
}

type ScriptDefinition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	FrequencyS  int64  `yaml:"frequencyS"`
	Script      string `yaml:"script"`
	AddExcludes bool   `yaml:"addExcludes,omitempty"`
	IsPreset    bool   `yaml:"-"`
}

type ScriptActions struct {
	ToDelete []*Script
	ToUpdate []*Script
	ToCreate []*Script
}

func IsNewRelicScript(scriptName string) bool {
	return strings.HasPrefix(scriptName, scriptPrefix)
}

func IsScriptForCluster(scriptName, clusterName string) bool {
	return IsNewRelicScript(scriptName) && strings.HasSuffix(scriptName, "-"+clusterName)
}

func GetActions(scriptDefinitions []*ScriptDefinition, currentScripts []*Script, config ScriptConfig) ScriptActions {
	definitions := make(map[string]ScriptDefinition)
	for _, definition := range scriptDefinitions {
		scriptName := getScriptName(definition.Name, config.ClusterName)
		frequencyS := getInterval(definition, config)
		if frequencyS > 0 {
			definitions[scriptName] = ScriptDefinition{
				Name:        scriptName,
				Description: definition.Description,
				FrequencyS:  frequencyS,
				Script:      templateScript(definition, config),
			}
		}
	}
	actions := ScriptActions{}
	for _, current := range currentScripts {
		if definition, present := definitions[current.Name]; present {
			if definition.Script != current.Script || definition.FrequencyS != current.FrequencyS || config.ClusterId != current.ClusterIds {
				actions.ToUpdate = append(actions.ToUpdate, &Script{
					ScriptDefinition: definition,
					ScriptId:         current.ScriptId,
					ClusterIds:       config.ClusterId,
				})
			}
			delete(definitions, current.Name)
		} else if IsNewRelicScript(current.Name) {
			actions.ToDelete = append(actions.ToDelete, current)
		}
	}
	for _, definition := range definitions {
		actions.ToCreate = append(actions.ToCreate, &Script{
			ScriptDefinition: definition,
			ClusterIds:       config.ClusterId,
		})
	}
	return actions
}

func getScriptName(scriptName string, clusterName string) string {
	return fmt.Sprintf("%s%s-%s", scriptPrefix, scriptName, clusterName)
}

func getInterval(definition *ScriptDefinition, config ScriptConfig) int64 {
	if definition.IsPreset {
		var interval int64
		if definition.Name == httpMetricsScript {
			interval = config.HttpMetricCollectInterval
		} else if definition.Name == httpSpansScript {
			interval = config.HttpSpanCollectInterval
		} else if definition.Name == jvmMetricsScript {
			interval = config.JvmCollectInterval
		} else if definition.Name == postgresqlSpansScript {
			interval = config.PostgresCollectInterval
		} else if definition.Name == mysqlSpansScript {
			interval = config.MysqlCollectInterval
		}
		if interval == 0 {
			interval = config.CollectInterval
		}
		return interval
	}
	if definition.FrequencyS == 0 {
		return config.CollectInterval
	}
	return definition.FrequencyS
}

func templateScript(definition *ScriptDefinition, config ScriptConfig) string {
	withIntegrationProvider := strings.Replace(definition.Script, "df.pixie = 'pixie'", "df.pixie = 'nr-pixie-integration'", -1)
	withClusterName := strings.Replace(withIntegrationProvider, "px.vizier_name()", "'"+config.ClusterName+"'", -1)
	lines := strings.Split(withClusterName, "\n")
	exportLineNumber := 0
	for i, line := range lines {
		if strings.Contains(line, "px.export(") {
			exportLineNumber = i
			break
		}
	}
	var finalLines []string
	finalLines = append(finalLines, lines[:exportLineNumber]...)
	if definition.IsPreset || definition.AddExcludes {
		finalLines = append(finalLines, "# New Relic integration filtering")
		finalLines = append(finalLines, getExcludeLines(config)...)
		if definition.IsPreset {
			finalLines = append(finalLines, getLimitLines(definition.Name, config)...)
		}
		finalLines = append(finalLines, "")
	}
	finalLines = append(finalLines, lines[exportLineNumber:]...)
	return strings.Join(finalLines, "\n")
}

func getExcludeLines(config ScriptConfig) []string {
	var lines []string
	if config.ExcludeNamespaces != "" {
		lines = append(lines, fmt.Sprintf("df = df[not px.regex_match('%s', df.namespace)]", config.ExcludeNamespaces))
	}
	if config.ExcludePods != "" {
		lines = append(lines, fmt.Sprintf("df = df[not px.regex_match('%s', df.pod)]", config.ExcludePods))
	}
	return lines
}

func getLimitLines(scriptName string, config ScriptConfig) []string {
	var lines []string
	if scriptName == httpSpansScript && config.HttpSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.HttpSpanLimit))
	} else if (scriptName == postgresqlSpansScript || scriptName == mysqlSpansScript) && config.DbSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.DbSpanLimit))
	}
	return lines
}
