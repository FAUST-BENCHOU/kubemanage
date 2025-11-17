package mcpclient

import (
	"fmt"
	"sync"

	appconfig "github.com/noovertime7/kubemanage/cmd/app/config"
	"github.com/noovertime7/kubemanage/pkg/logger"
	"go.uber.org/zap"
)

type Manager struct {
	mu          sync.RWMutex
	clients     map[string]*Client
	metas       map[string]ServerMeta
	defaultName string
}

func newManager(cfg *appconfig.MCPConfig) (*Manager, error) {
	configs, defaultName, err := BuildServerConfigs(cfg)
	if err != nil {
		return nil, err
	}

	mgr := &Manager{
		clients:     make(map[string]*Client, len(configs)),
		metas:       make(map[string]ServerMeta, len(configs)),
		defaultName: defaultName,
	}
	for _, conf := range configs {
		client, err := NewClient(conf)
		if err != nil {
			return nil, fmt.Errorf("初始化 MCP server %s 失败: %w", conf.Name, err)
		}
		mgr.clients[conf.Name] = client
		mgr.metas[conf.Name] = ServerMeta{
			Name:        conf.Name,
			DisplayName: conf.DisplayName,
			Description: conf.Description,
			DefaultTool: conf.DefaultTool,
			Image:       conf.Image,
			Homepage:    conf.Homepage,
			Tags:        append([]string(nil), conf.Tags...),
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
