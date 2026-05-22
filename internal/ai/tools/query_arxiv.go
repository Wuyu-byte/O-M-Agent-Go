package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/mark3labs/mcp-go/client"
)

/*
GetArxivMcpTool starts a local stdio MCP server provided by
https://github.com/blazickjp/arxiv-mcp-server and converts its tools
to Eino tools.
*/
func GetArxivMcpTool() ([]tool.BaseTool, error) {
	ctx := context.Background()
	command, err := g.Cfg().Get(ctx, "arxiv_mcp_command", "uv")
	if err != nil {
		return nil, err
	}
	args, err := g.Cfg().Get(ctx, "arxiv_mcp_args")
	if err != nil {
		return nil, err
	}
	cli, err := client.NewStdioMCPClient(command.String(), nil, args.Strings()...)
	if err != nil {
		return []tool.BaseTool{}, err
	}
	return getMcpTools(ctx, cli)
}
