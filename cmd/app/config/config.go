package config

var SysConfig *Config

type Config struct {
	Default DefaultOptions `mapstructure:"default"`
	Mysql   MysqlOptions   `mapstructure:"mysql"`
	CMDB    CMDBOptions    `mapstructure:"cmdb"`
	Log     LogConfig      `mapstructure:"log"`
	MCP     MCPConfig      `mapstructure:"mcp"`
}

type DefaultOptions struct {
	PodLogTailLine       string `mapstructure:"podLogTailLine"`
	ListenAddr           string `mapstructure:"listenAddr"`
	WebSocketListenAddr  string `mapstructure:"webSocketListenAddr"`
	JWTSecret            string `mapstructure:"JWTSecret"`
	ExpireTime           int64  `mapstructure:"expireTime"`
	KubernetesConfigFile string `mapstructure:"kubernetesConfigFile"`
}

type CMDBOptions struct {
	HostCheck HostCheck `mapstructure:"hostCheck"`
}

type HostCheck struct {
	HostCheckEnable   bool `mapstructure:"hostCheckEnable"`
	HostCheckDuration int  `mapstructure:"hostCheckDuration"`
	HostCheckTimeout  int  `mapstructure:"hostCheckTimeout"`
}

type MysqlOptions struct {
	Host         string `mapstructure:"host"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Port         string `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	MaxOpenConns int    `mapstructure:"maxOpenConns"`
	MaxLifetime  int    `mapstructure:"maxLifetime"`
	MaxIdleConns int    `mapstructure:"maxIdleConns"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

type MCPConfig struct {
	Enable                bool                `mapstructure:"enable"`
	ImplementationName    string              `mapstructure:"implementationName"`
	ImplementationVersion string              `mapstructure:"implementationVersion"`
	StartupTimeout        string              `mapstructure:"startupTimeout"`
	KeepAlive             string              `mapstructure:"keepAlive"`
	DefaultServer         string              `mapstructure:"defaultServer"`
	DefaultTool           string              `mapstructure:"defaultTool"`
	Command               string              `mapstructure:"command"` // 兼容旧配置
	Args                  []string            `mapstructure:"args"`    // 兼容旧配置
	Env                   map[string]string   `mapstructure:"env"`     // 兼容旧配置
	Servers               []MCPServerTemplate `mapstructure:"servers"`
}

type MCPServerTemplate struct {
	Name        string            `mapstructure:"name"`
	DisplayName string            `mapstructure:"displayName"`
	Description string            `mapstructure:"description"`
	Image       string            `mapstructure:"image"`
	Command     string            `mapstructure:"command"`
	Args        []string          `mapstructure:"args"`
	Env         map[string]string `mapstructure:"env"`
	DefaultTool string            `mapstructure:"defaultTool"`
	Homepage    string            `mapstructure:"homepage"`
	Tags        []string          `mapstructure:"tags"`
}
