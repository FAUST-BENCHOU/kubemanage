package mcpclient

import (
	"fmt"
	"strings"
	"time"

	appconfig "github.com/noovertime7/kubemanage/cmd/app/config"
)

const (
	defaultImplementationName    = "kubemanage-mcp-client"
	defaultImplementationVersion = "0.1.0"
	defaultStartupTimeout        = 20 * time.Second
	defaultKeepAlive             = 45 * time.Second
)

// Config 表示单个 MCP Server 的运行配置
type Config struct {
	Name                  string
	DisplayName           string
	Description           string
	Image                 string
	Homepage              string
	Tags                  []string
	ImplementationName    string
	ImplementationVersion string
	Command               string
	Args                  []string
	Env                   map[string]string
	StartupTimeout        time.Duration
	KeepAlive             time.Duration
	DefaultTool           string
}

// ServerMeta 为对外暴露的 Server 信息
type ServerMeta struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	DefaultTool string   `json:"default_tool"`
	Image       string   `json:"image"`
	Homepage    string   `json:"homepage,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// BuildServerConfigs 解析配置生成多个 Server Config 及默认 Server 名称
func BuildServerConfigs(cfg *appconfig.MCPConfig) ([]Config, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("未找到 mcp 配置")
	}

	servers := cfg.Servers
	if len(servers) == 0 && cfg.Command != "" {
		// 兼容旧配置，自动包装成一个 Server
		servers = []appconfig.MCPServerTemplate{
			{
				Name:        "default",
				DisplayName: "Default MCP Server",
				Description: "兼容旧版本配置生成的默认 MCP Server",
				Command:     cfg.Command,
				Args:        cfg.Args,
				Env:         cfg.Env,
				DefaultTool: cfg.DefaultTool,
			},
		}
	}

	if len(servers) == 0 {
		return nil, "", fmt.Errorf("未配置任何 MCP server")
	}

	startupTimeout := defaultStartupTimeout
	if cfg.StartupTimeout != "" {
		d, err := time.ParseDuration(cfg.StartupTimeout)
		if err != nil {
			return nil, "", fmt.Errorf("解析 mcp.startupTimeout 失败: %w", err)
		}
		startupTimeout = d
	}

	keepAlive := defaultKeepAlive
	if cfg.KeepAlive != "" {
		d, err := time.ParseDuration(cfg.KeepAlive)
		if err != nil {
			return nil, "", fmt.Errorf("解析 mcp.keepAlive 失败: %w", err)
		}
		keepAlive = d
	}

	implName := cfg.ImplementationName
	if implName == "" {
		implName = defaultImplementationName
	}
	implVersion := cfg.ImplementationVersion
	if implVersion == "" {
		implVersion = defaultImplementationVersion
	}

	var configs []Config
	for _, server := range servers {
		if server.Name == "" {
			return nil, "", fmt.Errorf("存在未命名的 MCP server，请检查配置")
		}
		command := server.Command
		if command == "" {
			command = cfg.Command
		}
		if command == "" {
			command = "docker"
		}

		args := server.Args
		if len(args) == 0 {
			if len(cfg.Args) > 0 {
				args = cfg.Args
			} else if server.Image != "" && strings.EqualFold(command, "docker") {
				args = []string{"run", "--rm", "-i", server.Image}
			}
		}
		if len(args) == 0 {
			return nil, "", fmt.Errorf("server %s 未配置启动参数", server.Name)
		}

		env := mergeEnv(cfg.Env, server.Env)
		defaultTool := server.DefaultTool
		if defaultTool == "" {
			defaultTool = cfg.DefaultTool
		}

		cfgCopy := Config{
			Name:                  server.Name,
			DisplayName:           firstNonEmpty(server.DisplayName, server.Name),
			Description:           server.Description,
			Image:                 firstNonEmpty(server.Image, extractImageFromArgs(args)),
			Homepage:              server.Homepage,
			Tags:                  append([]string(nil), server.Tags...),
			ImplementationName:    implName,
			ImplementationVersion: implVersion,
			Command:               command,
			Args:                  append([]string(nil), args...),
			Env:                   env,
			StartupTimeout:        startupTimeout,
			KeepAlive:             keepAlive,
			DefaultTool:           defaultTool,
		}
		configs = append(configs, cfgCopy)
	}

	defaultServer := cfg.DefaultServer
	if defaultServer == "" {
		defaultServer = configs[0].Name
	}

	return configs, defaultServer, nil
}

func mergeEnv(global, local map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range global {
		result[k] = v
	}
	for k, v := range local {
		result[k] = v
	}
	return result
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func extractImageFromArgs(args []string) string {
	for i := len(args) - 1; i >= 0; i-- {
		arg := args[i]
		if strings.HasPrefix(arg, "-") || strings.Contains(arg, "=") {
			continue
		}
		return arg
	}
	return ""
}
