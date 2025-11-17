package mcpclient

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/noovertime7/kubemanage/pkg/logger"
)

// Client 封装了与 MCP Server 的连接会话
type Client struct {
	cfg     Config
	impl    *mcp.Implementation
	client  *mcp.Client
	session *mcp.ClientSession
	cancel  context.CancelFunc

	mu     sync.Mutex
	closed bool
}

// NewClient 根据配置创建一个 MCP 客户端，但不会立即建立连接
func NewClient(cfg Config) (*Client, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("MCP 客户端未配置 command")
	}
	if cfg.ImplementationName == "" {
		cfg.ImplementationName = defaultImplementationName
	}
	if cfg.ImplementationVersion == "" {
		cfg.ImplementationVersion = defaultImplementationVersion
	}

	impl := &mcp.Implementation{
		Name:    cfg.ImplementationName,
		Version: cfg.ImplementationVersion,
	}

	opts := &mcp.ClientOptions{}
	if cfg.KeepAlive > 0 {
		opts.KeepAlive = cfg.KeepAlive
	}

	return &Client{
		cfg:    cfg,
		impl:   impl,
		client: mcp.NewClient(impl, opts),
	}, nil
}

// EnsureSession 确保存在一个有效的 MCP 会话，如果不存在则尝试连接
func (c *Client) EnsureSession(ctx context.Context) (*mcp.ClientSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("MCP 客户端已关闭")
	}
	if c.session != nil {
		return c.session, nil
	}

	connectCtx := ctx
	var cancel context.CancelFunc
	if c.cfg.StartupTimeout > 0 {
		connectCtx, cancel = context.WithTimeout(ctx, c.cfg.StartupTimeout)
	}
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	// 构建命令参数：如果是 docker run，需要将 env 转换为 -e 参数
	args := append([]string(nil), c.cfg.Args...)
	if len(c.cfg.Env) > 0 {
		// 检测是否是 docker run 命令
		isDockerRun := c.cfg.Command == "docker" && len(args) > 0 && args[0] == "run"
		if isDockerRun {
			// 对于 docker run，将环境变量转换为 -e 参数
			// 找到 "run" 之后的位置插入 -e 参数（通常在 --rm 或 -i 之前）
			insertPos := 1
			for i, arg := range args {
				if i > 0 && (arg == "--rm" || arg == "-i" || arg == "-d" || arg == "-it") {
					insertPos = i
					break
				}
			}
			// 构建 -e 参数列表
			envArgs := make([]string, 0, len(c.cfg.Env)*2)
			for k, v := range c.cfg.Env {
				envArgs = append(envArgs, "-e", fmt.Sprintf("%s=%s", k, v))
			}
			// 插入环境变量参数
			newArgs := make([]string, 0, len(args)+len(envArgs))
			newArgs = append(newArgs, args[:insertPos]...)
			newArgs = append(newArgs, envArgs...)
			newArgs = append(newArgs, args[insertPos:]...)
			args = newArgs
		} else {
			// 对于非 docker 命令，使用传统的环境变量方式
			// 这种情况会在 exec.CommandContext 之后通过 cmd.Env 设置
		}
	}

	cmd := exec.CommandContext(connectCtx, c.cfg.Command, args...)
	// 对于非 docker 命令，设置环境变量
	if len(c.cfg.Env) > 0 && !(c.cfg.Command == "docker" && len(c.cfg.Args) > 0 && c.cfg.Args[0] == "run") {
		env := os.Environ()
		for k, v := range c.cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}
	cmd.Stderr = os.Stderr

	session, err := c.client.Connect(connectCtx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("连接 MCP Server 失败: %w", err)
	}
	c.cancel = cancel
	cancel = nil
	c.session = session

	go c.waitSession(session)
	logger.LG.Info("MCP 会话已建立",
		zap.String("server", c.cfg.Name),
		zap.String("impl", c.cfg.ImplementationName),
		zap.String("command", c.cfg.Command),
	)

	return session, nil
}

// CallTool 调用指定工具并返回结果
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	if name == "" {
		return nil, fmt.Errorf("tool 名称不能为空")
	}

	session, err := c.EnsureSession(ctx)
	if err != nil {
		return nil, err
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("tool %s 返回空结果", name)
	}
	if result.IsError {
		return nil, fmt.Errorf("tool %s 调用失败: %s", name, TextContent(result.Content))
	}
	return result, nil
}

// ListTools 列出远端可用工具
func (c *Client) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	session, err := c.EnsureSession(ctx)
	if err != nil {
		return nil, err
	}
	return session.ListTools(ctx, nil)
}

// DefaultToolName 返回配置的默认工具
func (c *Client) DefaultToolName() string {
	return c.cfg.DefaultTool
}

// CallDefaultTool 使用默认工具名发起调用
func (c *Client) CallDefaultTool(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	if c.cfg.DefaultTool == "" {
		return nil, fmt.Errorf("未配置默认工具 defaultTool")
	}
	return c.CallTool(ctx, c.cfg.DefaultTool, args)
}

// Close 主动关闭会话
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.session == nil {
		return nil
	}
	err := c.session.Close()
	c.session = nil
	return err
}

func (c *Client) waitSession(cs *mcp.ClientSession) {
	if cs == nil {
		return
	}
	err := cs.Wait()

	c.mu.Lock()
	if c.session == cs {
		c.session = nil
		if c.cancel != nil {
			c.cancel()
			c.cancel = nil
		}
	}
	c.mu.Unlock()

	if err != nil {
		logger.LG.Warn("MCP 会话已结束", zap.Error(err))
	} else {
		logger.LG.Info("MCP 会话正常结束")
	}
}
