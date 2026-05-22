package chat_pipeline

import (
	"SuperBizAgent/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/gogf/gf/v2/frame/g"
)

func newReactAgentLambda(ctx context.Context) (lba *compose.Lambda, err error) {
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{}}
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = chatModelIns11
	//searchTool, err := newSearchTool(ctx)
	//if err != nil {
	//	return nil, err
	//}
	logMcpEnabled, err := g.Cfg().Get(ctx, "log_mcp_enabled", false)
	if err != nil {
		return nil, err
	}
	if logMcpEnabled.Bool() {
		mcpTool, err := tools.GetLogMcpTool()
		if err != nil {
			return nil, err
		}
		config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, mcpTool...)
	}
	arxivMcpEnabled, err := g.Cfg().Get(ctx, "arxiv_mcp_enabled", false)
	if err != nil {
		return nil, err
	}
	if arxivMcpEnabled.Bool() {
		arxivMcpTool, err := tools.GetArxivMcpTool()
		if err != nil {
			return nil, err
		}
		config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, arxivMcpTool...)
	}
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewPrometheusAlertsQueryTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewMysqlCrudTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewGetCurrentTimeTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewQueryInternalDocsTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewQueryGPUStatusTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewQueryPythonEnvTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewReadLabLogTool())

	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
