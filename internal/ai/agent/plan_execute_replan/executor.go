package plan_execute_replan

import (
	"SuperBizAgent/internal/ai/models"
	"SuperBizAgent/internal/ai/tools"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/gogf/gf/v2/frame/g"
)

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	var toolList []tool.BaseTool
	logMcpEnabled, err := g.Cfg().Get(ctx, "log_mcp_enabled", false)
	if err != nil {
		return nil, err
	}
	if logMcpEnabled.Bool() {
		// log
		mcpTool, err := tools.GetLogMcpTool()
		if err != nil {
			return nil, err
		}
		toolList = append(toolList, mcpTool...)
	}
	// alerts
	toolList = append(toolList, tools.NewPrometheusAlertsQueryTool())
	// file
	toolList = append(toolList, tools.NewQueryInternalDocsTool())
	// time
	toolList = append(toolList, tools.NewGetCurrentTimeTool())
	// lab environment
	toolList = append(toolList, tools.NewQueryGPUStatusTool())
	toolList = append(toolList, tools.NewQueryPythonEnvTool())
	toolList = append(toolList, tools.NewReadLabLogTool())
	execModel, err := models.OpenAIForDeepSeekV3Quick(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: execModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList,
			},
		},
		MaxIterations: 999999,
	})
}
