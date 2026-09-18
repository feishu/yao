package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	gouServer "github.com/yaoapp/gou/mcp/server"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/api"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: L("Model Context Protocol (MCP) server commands"),
	Long:  L("Model Context Protocol (MCP) server commands for AI assistants"),
}

var mcpStdioCmd = &cobra.Command{
	Use:   "stdio [group]",
	Short: L("Run MCP server over standard input/output (stdio) for local AI clients"),
	Long:  L("Run MCP server over standard input/output (stdio) for local AI clients such as Claude Desktop or Cursor"),
	Run: func(cmd *cobra.Command, args []string) {
		defer share.SessionStop()
		defer plugin.KillAll()

		defer func() {
			if err := exception.Catch(recover()); err != nil {
				fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
				os.Exit(1)
			}
		}()

		Boot()
		// 载入应用工程核心组件
		err := engine.Load(config.Conf, engine.LoadOption{Action: "run"})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Engine load failed: %v\n", err)
			os.Exit(1)
		}

		// 执行全量（API + Model）-> MCP 投影
		api.ProjectAllToMCP()

		group := "default"
		if len(args) > 0 && args[0] != "" {
			group = args[0]
		}

		srv := gouServer.GetServer(group)
		// 启动 Stdio 管道交互循环
		if err := srv.RunStdio(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "MCP Stdio server exited with error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	mcpCmd.AddCommand(mcpStdioCmd)
}
