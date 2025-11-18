package mcpclient

import (
	"fmt"
	"sync"

	appconfig "github.com/noovertime7/kubemanage/cmd/app/config"
	"github.com/noovertime7/kubemanage/pkg/logger"
	"github.com/noovertime7/kubemanage/runtime"
	"go.uber.org/zap"
)

var (
	defaultManager *Manager
	managerMu      sync.RWMutex
)

// InitFromConfig 根据全局配置初始化 MCP 管理器
func InitFromConfig(cfg appconfig.MCPConfig) error {
	if !cfg.Enable {
		logger.LG.Info("MCP 功能未启用，跳过初始化")
		return nil
	}

	mgr, err := newManager(&cfg)
	if err != nil {
		return err
	}

	managerMu.Lock()
	defaultManager = mgr
	managerMu.Unlock()

	runtime.CloserHandler.AddCloser(&managerCloser{manager: mgr})
	logger.LG.Info("MCP 管理器初始化完成", zap.Int("servers", len(mgr.clients)))
	return nil
}

// DefaultManager 返回默认管理器实例
func DefaultManager() *Manager {
	managerMu.RLock()
	defer managerMu.RUnlock()
	return defaultManager
}

// DefaultClient 返回默认 Server 对应的客户端
func DefaultClient() (*Client, error) {
	mgr := DefaultManager()
	if mgr == nil {
		return nil, fmt.Errorf("MCP 功能未启用")
	}
	return mgr.DefaultClient()
}

// ClientByName 根据 Server 名称获取客户端
func ClientByName(name string) (*Client, error) {
	mgr := DefaultManager()
	if mgr == nil {
		return nil, fmt.Errorf("MCP 功能未启用")
	}
	return mgr.Client(name)
}

// ListServers 返回所有可用的 MCP Server 元数据
func ListServers() []ServerMeta {
	mgr := DefaultManager()
	if mgr == nil {
		return nil
	}
	return mgr.ListServers()
}

type managerCloser struct {
	manager *Manager
}

func (m *managerCloser) Close() error {
	if m.manager == nil {
		return nil
	}
	return m.manager.Close()
}

func (m *managerCloser) HandlerErr(err error) {
	logger.LG.Error("关闭 MCP 管理器失败", zap.Error(err))
}
