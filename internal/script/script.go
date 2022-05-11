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
	ScriptId string
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

func GetActions(scriptDefinitions []*ScriptDefinition, currentScripts []*Script, config ScriptConfig) ScriptActions {
	definitions := make(map[string]ScriptDefinition)
	for _, definition := range scriptDefinitions {
		scriptName := getScriptName(definition.Name, config.ClusterName)
		definitions[scriptName] = ScriptDefinition{
			Name:        scriptName,
			Description: definition.Description,
			FrequencyS:  getInterval(definition, config),
			Script:      templateScript(definition, config),
		}
	}
	actions := ScriptActions{}
	for _, current := range currentScripts {
		if definition, present := definitions[current.Name]; present {
			if definition.Script != current.Script || definition.FrequencyS != current.FrequencyS {
				actions.ToUpdate = append(actions.ToUpdate, &Script{
					ScriptDefinition: definition,
					ScriptId:         current.ScriptId,
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
		})
	}
	return actions
}

func getScriptName(scriptName string, clusterName string) string {
	return fmt.Sprintf("%s%s-%s", scriptPrefix, scriptName, clusterName)
}

func getInterval(definition *ScriptDefinition, config ScriptConfig) int64 {
	if definition.IsPreset || definition.FrequencyS <= 0 {
		if definition.Name == httpMetricsScript {
			return config.HttpMetricCollectInterval
		} else if definition.Name == httpSpansScript {
			return config.HttpSpanCollectInterval
		} else if definition.Name == jvmMetricsScript {
			return config.JvmCollectInterval
		} else if definition.Name == postgresqlSpansScript {
			return config.PostgresCollectInterval
		} else if definition.Name == mysqlSpansScript {
			return config.MysqlCollectInterval
		}
		return config.CollectInterval
	}
	return definition.FrequencyS
}

func templateScript(definition *ScriptDefinition, config ScriptConfig) string {
	withClusterName := strings.Replace(definition.Script, "px.vizier_name()", "'"+config.ClusterName+"'", -1)
	if !definition.IsPreset && !definition.AddExcludes {
		return withClusterName
	}
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
	finalLines = append(finalLines, getFilterLines(definition.Name, config)...)
	finalLines = append(finalLines, lines[exportLineNumber:]...)
	return strings.Join(finalLines, "\n")
}

func getFilterLines(scriptName string, config ScriptConfig) []string {
	lines := []string{"# New Relic integration filtering"}
	if config.ExcludeNamespaces != "" {
		lines = append(lines, fmt.Sprintf("df = df[!px.regex_match('%s', df.namespace)]", config.ExcludeNamespaces))
	}
	if config.ExcludePods != "" {
		lines = append(lines, fmt.Sprintf("df = df[!px.regex_match('%s', df.pod)]", config.ExcludePods))
	}
	if scriptName == httpSpansScript && config.HttpSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.HttpSpanLimit))
	} else if (scriptName == postgresqlSpansScript || scriptName == mysqlSpansScript) && config.DbSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.DbSpanLimit))
	}
	return append(lines, "")
}
