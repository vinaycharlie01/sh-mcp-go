package config

import (
	"log/slog"
	"time"
)

// Config is the root application configuration.
type Config struct {
	Server        ServerConfig        `mapstructure:"server"      validate:"required"`
	Kubernetes    KubernetesConfig    `mapstructure:"kubernetes"`
	Helm          HelmConfig          `mapstructure:"helm"        validate:"required"`
	Storage       StorageConfig       `mapstructure:"storage"     validate:"required"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Security      SecurityConfig      `mapstructure:"security"`
	MCP           MCPConfig           `mapstructure:"mcp"         validate:"required"`
	Log           LogConfig           `mapstructure:"log"`
	Version       string              `mapstructure:"version"`
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Host            string        `mapstructure:"host"              validate:"required"`
	Port            int           `mapstructure:"port"              validate:"required,min=1,max=65535"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	TLSCertFile     string        `mapstructure:"tls_cert_file"`
	TLSKeyFile      string        `mapstructure:"tls_key_file"`
}

// KubernetesConfig controls Kubernetes client settings.
type KubernetesConfig struct {
	KubeconfigPath string        `mapstructure:"kubeconfig_path"`
	InCluster      bool          `mapstructure:"in_cluster"`
	QPS            float32       `mapstructure:"qps"`
	Burst          int           `mapstructure:"burst"`
	Timeout        time.Duration `mapstructure:"timeout"`
	DefaultNS      string        `mapstructure:"default_namespace"`
}

// HelmConfig controls Helm SDK behaviour.
type HelmConfig struct {
	RepositoryCache  string        `mapstructure:"repository_cache"  validate:"required"`
	RepositoryConfig string        `mapstructure:"repository_config" validate:"required"`
	RegistryConfig   string        `mapstructure:"registry_config"`
	PluginsDir       string        `mapstructure:"plugins_dir"`
	DefaultTimeout   time.Duration `mapstructure:"default_timeout"`
	MaxHistory       int           `mapstructure:"max_history"`
	Atomic           bool          `mapstructure:"atomic"`
	WaitForJobs      bool          `mapstructure:"wait_for_jobs"`
	// TLS settings for Helm repository connections.
	CAFile                string `mapstructure:"ca_file"`
	CertFile              string `mapstructure:"cert_file"`
	KeyFile               string `mapstructure:"key_file"`
	InsecureSkipTLSVerify bool   `mapstructure:"insecure_skip_tls_verify"`
	// OCI registry settings.
	PlainHTTP bool `mapstructure:"plain_http"`
}

// StorageConfig controls the persistence backend.
type StorageConfig struct {
	Driver   string         `mapstructure:"driver"   validate:"required,oneof=sqlite postgres"`
	SQLite   SQLiteConfig   `mapstructure:"sqlite"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

// SQLiteConfig is the SQLite storage configuration.
type SQLiteConfig struct {
	Path string `mapstructure:"path" validate:"required"`
}

// PostgresConfig is the PostgreSQL storage configuration.
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int    `mapstructure:"max_conns"`
	MinConns int    `mapstructure:"min_conns"`
}

// ObservabilityConfig controls telemetry.
type ObservabilityConfig struct {
	MetricsEnabled bool    `mapstructure:"metrics_enabled"`
	TracingEnabled bool    `mapstructure:"tracing_enabled"`
	OTLPEndpoint   string  `mapstructure:"otlp_endpoint"`
	ServiceName    string  `mapstructure:"service_name"`
	SamplingRate   float64 `mapstructure:"sampling_rate"`
}

// SecurityConfig controls security behaviour.
type SecurityConfig struct {
	EnableRBACValidation bool              `mapstructure:"enable_rbac_validation"`
	EnableSecretMasking  bool              `mapstructure:"enable_secret_masking"`
	AllowedNamespaces    []string          `mapstructure:"allowed_namespaces"`
	DeniedNamespaces     []string          `mapstructure:"denied_namespaces"`
	RequiredLabels       map[string]string `mapstructure:"required_labels"`
}

// MCPConfig controls the MCP server.
type MCPConfig struct {
	Transport string `mapstructure:"transport" validate:"required,oneof=stdio sse http"`
	SSEAddr   string `mapstructure:"sse_addr"`
	Name      string `mapstructure:"name"      validate:"required"`
	Version   string `mapstructure:"version"   validate:"required"`
}

// LogConfig controls structured logging.
type LogConfig struct {
	Level  string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"oneof=json text"`
}

// SlogLevel converts the log level string to slog.Level.
func (l LogConfig) SlogLevel() slog.Level {
	switch l.Level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
