package pixie

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gogo/protobuf/types"
	"github.com/newrelic/newrelic-pixie-integration/internal/script"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"px.dev/pxapi/proto/cloudpb"
	"px.dev/pxapi/proto/uuidpb"
	"px.dev/pxapi/utils"
	"strings"
)

const (
	newRelicPluginId = "new-relic"
	apiKeyConfig     = "api-key"
)

type Client struct {
	cloudAddr string
	ctx       context.Context

	grpcConn     *grpc.ClientConn
	pluginClient cloudpb.PluginServiceClient
}

func NewClient(ctx context.Context, apiKey string, cloudAddr string) (*Client, error) {
	c := &Client{
		cloudAddr: cloudAddr,
		ctx:       metadata.AppendToOutgoingContext(ctx, "pixie-api-key", apiKey),
	}

	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) init() error {
	isInternal := strings.ContainsAny(c.cloudAddr, "cluster.local")

	tlsConfig := &tls.Config{InsecureSkipVerify: isInternal}
	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.Dial(c.cloudAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}

	c.grpcConn = conn
	c.pluginClient = cloudpb.NewPluginServiceClient(conn)
	return nil
}

func (c *Client) GetNewRelicPlugin() (*cloudpb.Plugin, error) {
	req := &cloudpb.GetPluginsRequest{
		Kind: cloudpb.PK_RETENTION,
	}
	resp, err := c.pluginClient.GetPlugins(c.ctx, req)
	if err != nil {
		return nil, err
	}
	for _, plugin := range resp.Plugins {
		if plugin.Id == newRelicPluginId {
			return plugin, nil
		}
	}
	return nil, fmt.Errorf("the %s plugin could not be found", newRelicPluginId)
}

type NewRelicPluginConfig struct {
	LicenseKey string
	ExportUrl  string
}

func (c *Client) GetNewRelicPluginConfig() (*NewRelicPluginConfig, error) {
	req := &cloudpb.GetOrgRetentionPluginConfigRequest{
		PluginId: newRelicPluginId,
	}
	resp, err := c.pluginClient.GetOrgRetentionPluginConfig(c.ctx, req)
	if err != nil {
		return nil, err
	}
	exportUrl := resp.CustomExportUrl
	if exportUrl == "" {
		exportUrl, err = c.getDefaultNewRelicExportUrl()
		if err != nil {
			return nil, err
		}
	}
	return &NewRelicPluginConfig{
		LicenseKey: resp.Configs[apiKeyConfig],
		ExportUrl:  exportUrl,
	}, nil
}

func (c *Client) getDefaultNewRelicExportUrl() (string, error) {
	req := &cloudpb.GetRetentionPluginInfoRequest{
		PluginId: newRelicPluginId,
	}
	info, err := c.pluginClient.GetRetentionPluginInfo(c.ctx, req)
	if err != nil {
		return "", err
	}
	return info.DefaultExportURL, nil
}

func (c *Client) EnableNewRelicPlugin(config *NewRelicPluginConfig, version string) error {
	req := &cloudpb.UpdateRetentionPluginConfigRequest{
		PluginId: newRelicPluginId,
		Configs: map[string]string{
			apiKeyConfig: config.LicenseKey,
		},
		Enabled:         &types.BoolValue{Value: true},
		Version:         &types.StringValue{Value: version},
		CustomExportUrl: &types.StringValue{Value: config.ExportUrl},
		InsecureTLS:     &types.BoolValue{Value: false},
		DisablePresets:  &types.BoolValue{Value: true},
	}
	_, err := c.pluginClient.UpdateRetentionPluginConfig(c.ctx, req)
	return err
}

func (c *Client) GetPresetScripts() ([]*script.ScriptDefinition, error) {
	resp, err := c.pluginClient.GetRetentionScripts(c.ctx, &cloudpb.GetRetentionScriptsRequest{})
	if err != nil {
		return nil, err
	}
	var l []*script.ScriptDefinition
	for _, s := range resp.Scripts {
		if s.IsPreset {
			sd, err := c.getScriptDefinition(s)
			if err != nil {
				return nil, err
			}
			l = append(l, sd)
		}
	}
	return l, nil
}

func (c *Client) GetClusterScripts(clusterId, clusterName string) ([]*script.Script, error) {
	resp, err := c.pluginClient.GetRetentionScripts(c.ctx, &cloudpb.GetRetentionScriptsRequest{})
	if err != nil {
		return nil, err
	}
	var l []*script.Script
	for _, s := range resp.Scripts {
		if script.IsNewRelicScript(s.ScriptName) && (script.IsScriptForCluster(s.ScriptName, clusterName) ||
			(len(s.ClusterIDs) == 1 && utils.ProtoToUUIDStr(s.ClusterIDs[0]) == clusterId)) {
			sd, err := c.getScriptDefinition(s)
			if err != nil {
				return nil, err
			}
			l = append(l, &script.Script{
				ScriptDefinition: *sd,
				ScriptId:         utils.ProtoToUUIDStr(s.ScriptID),
				ClusterIds:       getClusterIdsAsString(s.ClusterIDs),
			})
		}
	}
	return l, nil
}

func getClusterIdsAsString(clusterIDs []*uuidpb.UUID) string {
	scriptClusterId := ""
	for i, id := range clusterIDs {
		if i > 0 {
			scriptClusterId = scriptClusterId + ","
		}
		scriptClusterId = scriptClusterId + utils.ProtoToUUIDStr(id)
	}
	return scriptClusterId
}

func (c *Client) getScriptDefinition(s *cloudpb.RetentionScript) (*script.ScriptDefinition, error) {
	resp, err := c.pluginClient.GetRetentionScript(c.ctx, &cloudpb.GetRetentionScriptRequest{ID: s.ScriptID})
	if err != nil {
		return nil, err
	}
	return &script.ScriptDefinition{
		Name:        s.ScriptName,
		Description: s.Description,
		FrequencyS:  s.FrequencyS,
		Script:      resp.Contents,
		IsPreset:    s.IsPreset,
	}, nil
}

func (c *Client) AddDataRetentionScript(clusterId string, scriptName string, description string, frequencyS int64, contents string) error {
	req := &cloudpb.CreateRetentionScriptRequest{
		ScriptName:  scriptName,
		Description: description,
		FrequencyS:  frequencyS,
		Contents:    contents,
		ClusterIDs:  []*uuidpb.UUID{utils.ProtoFromUUIDStrOrNil(clusterId)},
		PluginId:    newRelicPluginId,
	}
	_, err := c.pluginClient.CreateRetentionScript(c.ctx, req)
	return err
}

func (c *Client) UpdateDataRetentionScript(clusterId string, scriptId string, scriptName string, description string, frequencyS int64, contents string) error {
	req := &cloudpb.UpdateRetentionScriptRequest{
		ID:          utils.ProtoFromUUIDStrOrNil(scriptId),
		ScriptName:  &types.StringValue{Value: scriptName},
		Description: &types.StringValue{Value: description},
		Enabled:     &types.BoolValue{Value: true},
		FrequencyS:  &types.Int64Value{Value: frequencyS},
		Contents:    &types.StringValue{Value: contents},
		ClusterIDs:  []*uuidpb.UUID{utils.ProtoFromUUIDStrOrNil(clusterId)},
	}
	_, err := c.pluginClient.UpdateRetentionScript(c.ctx, req)
	return err
}

func (c *Client) DeleteDataRetentionScript(scriptId string) error {
	req := &cloudpb.DeleteRetentionScriptRequest{
		ID: utils.ProtoFromUUIDStrOrNil(scriptId),
	}
	_, err := c.pluginClient.DeleteRetentionScript(c.ctx, req)
	return err
}
