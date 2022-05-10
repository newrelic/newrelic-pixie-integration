package script

import (
	"fmt"
	"strings"
)

const (
	scriptPrefix = "nri-"
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
	ScriptId  string
	ClusterId string
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
					ClusterId:        config.ClusterId,
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
		if definition.Name == "http_metrics" {
			return config.HttpMetricCollectInterval
		} else if definition.Name == "http_spans" {
			return config.HttpSpanCollectInterval
		} else if definition.Name == "jvm_metrics" {
			return config.JvmCollectInterval
		} else if definition.Name == "postgres" {
			return config.PostgresCollectInterval
		} else if definition.Name == "mysql" {
			return config.MysqlCollectInterval
		}
		return config.CollectInterval
	}
	return definition.FrequencyS
}

func templateScript(definition *ScriptDefinition, config ScriptConfig) string {
	withClusterName := strings.Replace(definition.Script, "px.vizier_name()", config.ClusterName, -1)
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
	finalLines := append(lines[:exportLineNumber], getFilterLines(definition.Name, config)...)
	finalLines = append(finalLines, lines[exportLineNumber:]...)
	return strings.Join(lines, "\n")
}

func getFilterLines(scriptName string, config ScriptConfig) []string {
	lines := []string{"# New Relic integration filtering"}
	if config.ExcludeNamespaces != "" {
		lines = append(lines, fmt.Sprintf("df = df[!px.regex_match('%s', df.namespace)\n]", config.ExcludeNamespaces))
	}
	if config.ExcludePods != "" {
		lines = append(lines, fmt.Sprintf("df = df[!px.regex_match('%s', df.pod)\n]", config.ExcludePods))
	}
	if scriptName == "http_spans" && config.HttpSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.HttpSpanLimit))
	} else if (scriptName == "postgres" || scriptName == "mysql") && config.DbSpanLimit > 0 {
		lines = append(lines, fmt.Sprintf("df = df.head(%v)", config.DbSpanLimit))
	}
	return append(lines, "")
}
