package script

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testScriptHead = `
import px
df = px.DataFrame(table='http_events', start_time=px.plugin.start_time)
df.namespace = df.ctx['namespace']
df.status_code = df.resp_status

df = df.groupby(['status_code', 'namespace']).agg(
    latency_min=('latency', px.min),
    latency_max=('latency', px.max),
    latency_sum=('latency', px.sum),
    latency_count=('latency', px.count),
    time_=('time_', px.max),
)

df.latency_min = df.latency_min / 1000000

df.cluster_name = %s
df.cluster_id = px.vizier_id()
df.pixie = 'pixie'
`
	testScriptTail = `
%spx.export(
  df, px.otel.Data(
    resource={%s
      'k8s.namespace.name': df.namespace,
      'px.cluster.id': df.cluster_id,
      'k8s.cluster.name': df.cluster_name,
      'instrumentation.provider': df.pixie,
    },
    data=[
      px.otel.metric.Summary(
        name='http.server.duration',
        description='measures the duration of the inbound HTTP request',
        # Unit is not supported yet
        # unit='ms',
        count=df.latency_count,
        sum=df.latency_sum,
        quantile_values={
          0.0: df.latency_min,
          1.0: df.latency_max,
        },
        attributes={
          'http.status_code': df.status_code,
        },
    )],
  ),
)
`
)

var testScript = fmt.Sprintf(testScriptHead, "px.vizier_name()") + fmt.Sprintf(testScriptTail, "", "")
var sourceColLine = "df.source = 'nr-pixie-integration'\n"
var sourceAttr = "'px.source': df.source,"

func getTemplatedScript(clusterName string, filter ...string) string {
	return fmt.Sprintf(testScriptHead, "'"+clusterName+"'") + strings.Join(filter, "\n") + fmt.Sprintf(testScriptTail, sourceColLine, sourceAttr)
}

func TestIsNewRelicScript(t *testing.T) {
	assert.True(t, IsNewRelicScript("nri-script-cluster"))
	assert.False(t, IsNewRelicScript("not-nri-script"))
}

func TestIsScriptForCluster(t *testing.T) {
	assert.True(t, IsScriptForCluster("nri-HTPT Metrics-test-cluster", "test-cluster"))
	assert.False(t, IsScriptForCluster("nri-HTPT Metrics-test-cluster", "new-cluster"))
}

func TestGetScriptName(t *testing.T) {
	assert.Equal(t, "nri-HTTP Metrics-test-cluster", getScriptName("HTTP Metrics", "test-cluster"))
}
func TestGetIntervalCustomScript(t *testing.T) {
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "custom script",
		FrequencyS: 0,
		IsPreset:   false,
	}, ScriptConfig{
		CollectInterval: 10,
	}))
	assert.Equal(t, int64(-1), getInterval(&ScriptDefinition{
		Name:       "custom script",
		FrequencyS: -1,
		IsPreset:   false,
	}, ScriptConfig{
		CollectInterval: 10,
	}))
	assert.Equal(t, int64(20), getInterval(&ScriptDefinition{
		Name:       "custom script",
		FrequencyS: 20,
		IsPreset:   false,
	}, ScriptConfig{
		CollectInterval: 10,
	}))
}

func TestGetIntervalPresetScript(t *testing.T) {
	assert.Equal(t, int64(30), getInterval(&ScriptDefinition{
		Name:       "New Preset",
		FrequencyS: 30,
		IsPreset:   true,
	}, ScriptConfig{
		CollectInterval: 100,
	}))
}

func TestGetActions(t *testing.T) {
	// No definitions, no scripts, nothing to do
	actions := GetActions([]*ScriptDefinition{}, []*Script{}, ScriptConfig{})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 0, len(actions.ToCreate))

	// No definitions, only a non-New Relic script, nothing to do
	actions = GetActions([]*ScriptDefinition{}, []*Script{
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name: "other-script",
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
	}, ScriptConfig{
		ClusterId: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
	})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 0, len(actions.ToCreate))

	// No definitions, 1 (outdated) New Relic script, delete the outdated script
	actions = GetActions([]*ScriptDefinition{}, []*Script{
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name: "nri-script-another-cluster",
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
	}, ScriptConfig{
		ClusterId: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
	})
	assert.Equal(t, 1, len(actions.ToDelete))
	assert.Equal(t, "06906e7e-c684-4858-9fa1-e0bf552b40a6", actions.ToDelete[0].ScriptId)
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 0, len(actions.ToCreate))

	// 1 inactive (negative frequencyS) preset script, no current scripts, nothing to do
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "Http Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  -1,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    false,
		},
	}, []*Script{}, ScriptConfig{})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 0, len(actions.ToCreate))

	// 1 preset script, no current scripts, create the script
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
	}, []*Script{}, ScriptConfig{
		ClusterName:     "test-cluster",
		ClusterId:       "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		CollectInterval: 10,
	})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 1, len(actions.ToCreate))

	assert.Equal(t, "nri-HTTP Metrics-test-cluster", actions.ToCreate[0].Name)
	assert.Equal(t, "This script sends HTTP metrics to New Relic's OTel endpoint.", actions.ToCreate[0].Description)
	assert.Equal(t, int64(10), actions.ToCreate[0].FrequencyS)
	assert.Equal(t, getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""), actions.ToCreate[0].Script)

	// don't update exact same script
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
	}, []*Script{
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name:        "nri-HTTP Metrics-test-cluster",
				Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
				FrequencyS:  10,
				Script:      getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""),
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
	}, ScriptConfig{
		ClusterName:     "test-cluster",
		ClusterId:       "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		CollectInterval: 10,
	})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 0, len(actions.ToUpdate))
	assert.Equal(t, 0, len(actions.ToCreate))

	// update script with different Script
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
	}, []*Script{
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name:        "nri-HTTP Metrics-test-cluster",
				Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
				FrequencyS:  10,
				Script:      getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""),
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
	}, ScriptConfig{
		ClusterName:       "test-cluster",
		ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		CollectInterval:   10,
		ExcludeNamespaces: "mynamespace.*",
	})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 1, len(actions.ToUpdate))
	assert.Equal(t, getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('mynamespace.*', df.namespace)]", ""), actions.ToUpdate[0].Script)
	assert.Equal(t, 0, len(actions.ToCreate))

	// update script with different ClusterId
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
	}, []*Script{
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name:        "nri-HTTP Metrics-test-cluster",
				Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
				FrequencyS:  10,
				Script:      getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""),
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "",
		},
	}, ScriptConfig{
		ClusterName:     "test-cluster",
		ClusterId:       "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		CollectInterval: 10,
	})
	assert.Equal(t, 0, len(actions.ToDelete))
	assert.Equal(t, 1, len(actions.ToUpdate))
	assert.Equal(t, "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484", actions.ToUpdate[0].ClusterIds)
	assert.Equal(t, 0, len(actions.ToCreate))

	// Full blown example with outdated, inactive and new scripts
	actions = GetActions([]*ScriptDefinition{
		&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
		&ScriptDefinition{
			Name:        "HTTP Spans",
			Description: "This script sends HTTP spans to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
		&ScriptDefinition{
			Name:        "JVM Metrics",
			Description: "This script sends JVM metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		},
		&ScriptDefinition{
			Name:        "Custom Script",
			Description: "My custom script",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    false,
		},
	}, []*Script{
		// outdated: different cluster name in script name
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name: "nri-HTTP Metrics-another-cluster",
			},
			ScriptId:   "06906e7e-c684-4858-9fa1-e0bf552b40a6",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
		// outdated: spans are now disabled
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name: "nri-HTTP Spans-test-cluster",
			},
			ScriptId:   "cc6455ca-e12e-4a1d-b81c-ecc97a3d44cf",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
		// outdated: missing filter on mynamespace
		&Script{
			ScriptDefinition: ScriptDefinition{
				Name:        "nri-JVM Metrics-test-cluster",
				Description: "This script sends JVM metrics to New Relic's OTel endpoint.",
				FrequencyS:  20,
				Script:      testScript,
			},
			ScriptId:   "4e4e51b2-86a8-4d57-a2a9-6771d15afcae",
			ClusterIds: "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		},
	}, ScriptConfig{
		ClusterName:       "test-cluster",
		ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
		CollectInterval:   20,
		ExcludeNamespaces: "mynamespace.*",
	})
	assert.Equal(t, 1, len(actions.ToDelete))
	assert.Equal(t, "06906e7e-c684-4858-9fa1-e0bf552b40a6", actions.ToDelete[0].ScriptId)

	assert.Equal(t, 2, len(actions.ToUpdate))
	assert.Equal(t, "cc6455ca-e12e-4a1d-b81c-ecc97a3d44cf", actions.ToUpdate[0].ScriptId)
	assert.Equal(t, int64(10), actions.ToUpdate[0].FrequencyS)
	assert.Equal(t, "4e4e51b2-86a8-4d57-a2a9-6771d15afcae", actions.ToUpdate[1].ScriptId)
	assert.Equal(t, int64(10), actions.ToUpdate[1].FrequencyS)
	assert.Equal(t, getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('mynamespace.*', df.namespace)]", ""), actions.ToUpdate[0].Script)

	assert.Equal(t, 2, len(actions.ToCreate))
	var httpMetricsScript, customScript *Script
	if actions.ToCreate[0].Name == "nri-HTTP Metrics-test-cluster" {
		httpMetricsScript = actions.ToCreate[0]
		customScript = actions.ToCreate[1]
	} else {
		httpMetricsScript = actions.ToCreate[1]
		customScript = actions.ToCreate[0]
	}

	assert.Equal(t, "nri-HTTP Metrics-test-cluster", httpMetricsScript.Name)
	assert.Equal(t, "This script sends HTTP metrics to New Relic's OTel endpoint.", httpMetricsScript.Description)
	assert.Equal(t, int64(10), httpMetricsScript.FrequencyS)
	assert.Equal(t, getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('mynamespace.*', df.namespace)]", ""), httpMetricsScript.Script)

	assert.Equal(t, "nri-Custom Script-test-cluster", customScript.Name)
	assert.Equal(t, "My custom script", customScript.Description)
	assert.Equal(t, int64(10), customScript.FrequencyS)
	assert.Equal(t, getTemplatedScript("test-cluster", ""), customScript.Script)
}

func TestTemplateScript(t *testing.T) {
	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""),
		templateScript(&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			HttpSpanLimit:     0,
			DbSpanLimit:       0,
			ExcludePods:       "",
			ExcludeNamespaces: "",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('.*mypod.*', df.pod)]", ""),
		templateScript(&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			HttpSpanLimit:     0,
			DbSpanLimit:       0,
			ExcludePods:       ".*mypod.*",
			ExcludeNamespaces: "",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('.*mynamespace.*', df.namespace)]", "df = df[not px.regex_match('.*mypod.*', df.pod)]", ""),
		templateScript(&ScriptDefinition{
			Name:        "HTTP Metrics",
			Description: "This script sends HTTP metrics to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			HttpSpanLimit:     0,
			DbSpanLimit:       0,
			ExcludePods:       ".*mypod.*",
			ExcludeNamespaces: ".*mynamespace.*",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('.*mynamespace.*', df.namespace)]", "df = df[not px.regex_match('.*mypod.*', df.pod)]", "df = df.head(100)", ""),
		templateScript(&ScriptDefinition{
			Name:        "HTTP Spans",
			Description: "This script sends HTTP spans to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			HttpSpanLimit:     100,
			DbSpanLimit:       0,
			ExcludePods:       ".*mypod.*",
			ExcludeNamespaces: ".*mynamespace.*",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('.*mynamespace.*', df.namespace)]", "df = df.head(200)", ""),
		templateScript(&ScriptDefinition{
			Name:        "MySQL Spans",
			Description: "This script sends MySQL spans to New Relic's OTel endpoint.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    true,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			HttpSpanLimit:     0,
			DbSpanLimit:       200,
			ExcludePods:       "",
			ExcludeNamespaces: ".*mynamespace.*",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", ""),
		templateScript(&ScriptDefinition{
			Name:        "My script",
			Description: "This is my script.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: false,
			IsPreset:    false,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			ExcludePods:       "",
			ExcludeNamespaces: ".*mynamespace.*",
		}))

	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", "df = df[not px.regex_match('.*mynamespace.*', df.namespace)]", ""),
		templateScript(&ScriptDefinition{
			Name:        "My script",
			Description: "This is my script.",
			FrequencyS:  10,
			Script:      testScript,
			AddExcludes: true,
			IsPreset:    false,
		}, ScriptConfig{
			ClusterName:       "test-cluster",
			ClusterId:         "91cb2c1d-e6fd-4fb9-9d2f-8358895bf484",
			ExcludePods:       "",
			ExcludeNamespaces: ".*mynamespace.*",
		}))
}
