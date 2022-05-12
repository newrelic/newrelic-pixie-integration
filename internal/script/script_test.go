package script

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
px.export(
  df, px.otel.Data(
    resource={
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

var testScript = fmt.Sprintf(testScriptHead, "px.vizier_name()") + testScriptTail

func getTemplatedScript(clusterName string, filter ...string) string {
	return fmt.Sprintf(testScriptHead, "'"+clusterName+"'") + strings.Join(filter, "\n") + testScriptTail
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
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "HTTP Metrics",
		FrequencyS: 0,
		IsPreset:   true,
	}, ScriptConfig{
		HttpMetricCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "HTTP Metrics",
		FrequencyS: 20,
		IsPreset:   true,
	}, ScriptConfig{
		HttpMetricCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "HTTP Metrics",
		FrequencyS: -1,
		IsPreset:   true,
	}, ScriptConfig{
		HttpMetricCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "HTTP Metrics",
		FrequencyS: 100,
		IsPreset:   true,
	}, ScriptConfig{
		HttpMetricCollectInterval: 0,
		CollectInterval:           10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "New Preset",
		FrequencyS: 100,
		IsPreset:   true,
	}, ScriptConfig{
		CollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "HTTP Spans",
		FrequencyS: 0,
		IsPreset:   true,
	}, ScriptConfig{
		HttpSpanCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "JVM Metrics",
		FrequencyS: 0,
		IsPreset:   true,
	}, ScriptConfig{
		JvmCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "MySQL Spans",
		FrequencyS: 0,
		IsPreset:   true,
	}, ScriptConfig{
		MysqlCollectInterval: 10,
	}))
	assert.Equal(t, int64(10), getInterval(&ScriptDefinition{
		Name:       "PostgreSQL Spans",
		FrequencyS: 0,
		IsPreset:   true,
	}, ScriptConfig{
		PostgresCollectInterval: 10,
	}))
}

func TestGetActions(t *testing.T) {
	// TODO Write GetAction tests here.
}

func TestTemplateScript(t *testing.T) {
	assert.Equal(t,
		getTemplatedScript("test-cluster", "", "# New Relic integration filtering", ""),
		templateScript(&ScriptDefinition{
			Name:        "Http Metrics",
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
			Name:        "Http Metrics",
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
