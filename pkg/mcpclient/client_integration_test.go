//go:build integration

package mcpclient

import (
	"context"
	"os"
	"testing"
	"time"

	appconfig "github.com/noovertime7/kubemanage/cmd/app/config"
	"github.com/noovertime7/kubemanage/pkg/logger"
	"go.uber.org/zap"
)

func TestScholarlyIntegration(t *testing.T) {
	if os.Getenv("RUN_MCP_INTEGRATION") != "1" {
		t.Skip("set RUN_MCP_INTEGRATION=1 to run")
	}

	cfg := appconfig.MCPConfig{
		Enable:                true,
		ImplementationName:    "kubemanage-mcp-client-test",
		ImplementationVersion: "0.1.0",
		StartupTimeout:        "90s",
		KeepAlive:             "45s",
		DefaultServer:         "scholarly",
		DefaultTool:           "search-arxiv",
		Servers: []appconfig.MCPServerTemplate{
			{
				Name:    "scholarly",
				Command: "docker",
				Args: []string{
					"run", "--rm", "-i",
					"ghcr.io/metorial/mcp-container--adityak74--mcp-scholarly--mcp-scholarly",
				},
				DefaultTool: "search-arxiv",
			},
		},
	}

	configs, _, err := BuildServerConfigs(&cfg)
	if err != nil {
		t.Fatalf("build config failed: %v", err)
	}

	logger.LG = zap.NewNop()

	client, err := NewClient(configs[0])
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if _, err := client.EnsureSession(ctx); err != nil {
		t.Fatalf("ensure session failed: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatalf("no tools returned")
	}

	result, err := client.CallTool(ctx, "search-arxiv", map[string]any{
		"keyword": "model context protocol",
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if text := TextContent(result.Content); text == "" {
		t.Fatalf("empty text content: %+v", result.Content)
	}
}
