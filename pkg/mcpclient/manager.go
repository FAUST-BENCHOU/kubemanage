package mcpclient

import (
	"fmt"
	"strings"
	"sync"
	"time"

	appconfig "github.com/noovertime7/kubemanage/cmd/app/config"
	"github.com/noovertime7/kubemanage/pkg/logger"
	"go.uber.org/zap"
)

type Manager struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	metas          map[string]ServerMeta
	defaultName    string
	implName       string
	implVersion    string
	startupTimeout time.Duration
	keepAlive      time.Duration
	globalEnv      map[string]string
}

func newManager(cfg *appconfig.MCPConfig) (*Manager, error) {
	configs, defaultName, err := BuildServerConfigs(cfg)
	if err != nil {
		return nil, err
	}

	globalEnv := make(map[string]string)
	for k, v := range cfg.Env {
		globalEnv[k] = v
	}

	mgr := &Manager{
		clients:        make(map[string]*Client, len(configs)),
		metas:          make(map[string]ServerMeta, len(configs)),
		defaultName:    defaultName,
		implName:       configs[0].ImplementationName,
		implVersion:    configs[0].ImplementationVersion,
		startupTimeout: configs[0].StartupTimeout,
		keepAlive:      configs[0].KeepAlive,
		globalEnv:      globalEnv,
	}
	for _, conf := range configs {
		if err := mgr.addConfig(conf, false); err != nil {
			return nil, err
		}
	}
	return mgr, nil
}

func (m *Manager) ListServers() []ServerMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ServerMeta, 0, len(m.metas))
	for _, meta := range m.metas {
		result = append(result, meta)
	}
	return result
}

func (m *Manager) Client(name string) (*Client, error) {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("未找到名为 %s 的 MCP server", name)
	}
	return client, nil
}

func (m *Manager) DefaultClient() (*Client, error) {
	return m.Client(m.defaultName)
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			logger.LG.Warn("关闭 MCP server 客户端失败", zap.String("server", name), zap.Error(err))
		}
	}
	return nil
}

func (m *Manager) HandlerErr(err error) {
	logger.LG.Error("关闭 MCP 管理器失败", zap.Error(err))
}

type CreateServerOptions struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Image       string            `json:"image"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	DefaultTool string            `json:"default_tool"`
	Homepage    string            `json:"homepage"`
	Tags        []string          `json:"tags"`
	SetDefault  bool              `json:"set_default"`
}

func (m *Manager) AddServer(opts *CreateServerOptions) (ServerMeta, error) {
	if opts == nil {
		return ServerMeta{}, fmt.Errorf("无效的 server 参数")
	}
	if strings.TrimSpace(opts.Name) == "" {
		return ServerMeta{}, fmt.Errorf("server 名称不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[opts.Name]; exists {
		return ServerMeta{}, fmt.Errorf("server %s 已存在", opts.Name)
	}

	cfg, err := m.buildConfigFromOptions(opts)
	if err != nil {
		return ServerMeta{}, err
	}

	if err := m.addConfig(cfg, opts.SetDefault); err != nil {
		return ServerMeta{}, err
	}

	meta := m.metas[cfg.Name]
	return meta, nil
}

func (m *Manager) buildConfigFromOptions(opts *CreateServerOptions) (Config, error) {
	command := opts.Command
	if command == "" {
		command = "docker"
	}

	args := append([]string(nil), opts.Args...)
	if len(args) == 0 && opts.Image != "" && strings.EqualFold(command, "docker") {
		args = []string{"run", "--rm", "-i", opts.Image}
	}
	if len(args) == 0 {
		return Config{}, fmt.Errorf("server %s 缺少启动参数 args", opts.Name)
	}

	env := mergeEnv(m.globalEnv, opts.Env)

	return Config{
		Name:                  opts.Name,
		DisplayName:           firstNonEmpty(opts.DisplayName, opts.Name),
		Description:           opts.Description,
		Image:                 firstNonEmpty(opts.Image, extractImageFromArgs(args)),
		Homepage:              opts.Homepage,
		Tags:                  append([]string(nil), opts.Tags...),
		ImplementationName:    m.implName,
		ImplementationVersion: m.implVersion,
		Command:               command,
		Args:                  args,
		Env:                   env,
		StartupTimeout:        m.startupTimeout,
		KeepAlive:             m.keepAlive,
		DefaultTool:           opts.DefaultTool,
	}, nil
}

func (m *Manager) addConfig(conf Config, setDefault bool) error {
	client, err := NewClient(conf)
	if err != nil {
		return fmt.Errorf("初始化 MCP server %s 失败: %w", conf.Name, err)
	}
	m.clients[conf.Name] = client
	m.metas[conf.Name] = ServerMeta{
		Name:        conf.Name,
		DisplayName: conf.DisplayName,
		Description: conf.Description,
		DefaultTool: conf.DefaultTool,
		Image:       conf.Image,
		Homepage:    conf.Homepage,
		Tags:        append([]string(nil), conf.Tags...),
	}
	if setDefault {
		m.defaultName = conf.Name
	}
	return nil
}
