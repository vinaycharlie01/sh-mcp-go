package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Loader loads and validates application configuration.
type Loader struct {
	v        *viper.Viper
	validate *validator.Validate
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/sh-mcp-go/")
	v.AddConfigPath("$HOME/.sh-mcp-go/")
	v.AddConfigPath("./configs/")
	v.AddConfigPath(".")

	v.SetEnvPrefix("SHMCP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	return &Loader{
		v:        v,
		validate: validator.New(),
	}
}

// Load reads the configuration from file and environment.
func (l *Loader) Load() (*Config, error) {
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := l.validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file path.
func (l *Loader) LoadFromFile(path string) (*Config, error) {
	l.v.SetConfigFile(path)
	return l.Load()
}

// Watch sets up a hot-reload callback invoked when the config file changes.
func (l *Loader) Watch(onChange func(*Config, error)) {
	l.v.OnConfigChange(func(_ fsnotify.Event) {
		cfg, err := l.Load()
		onChange(cfg, err)
	})
	l.v.WatchConfig()
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)
	v.SetDefault("server.shutdown_timeout", 15*time.Second)

	v.SetDefault("kubernetes.qps", float32(50))
	v.SetDefault("kubernetes.burst", 100)
	v.SetDefault("kubernetes.timeout", 30*time.Second)
	v.SetDefault("kubernetes.default_namespace", "default")

	v.SetDefault("helm.repository_cache", "/tmp/helm/cache")
	v.SetDefault("helm.repository_config", "/tmp/helm/repositories.yaml")
	v.SetDefault("helm.default_timeout", 5*time.Minute)
	v.SetDefault("helm.max_history", 10)
	v.SetDefault("helm.atomic", true)
	v.SetDefault("helm.wait_for_jobs", true)

	v.SetDefault("storage.driver", "sqlite")
	v.SetDefault("storage.sqlite.path", "/var/lib/sh-mcp-go/state.db")

	v.SetDefault("observability.metrics_enabled", true)
	v.SetDefault("observability.tracing_enabled", false)
	v.SetDefault("observability.service_name", "sh-mcp-go")
	v.SetDefault("observability.sampling_rate", 0.1)

	v.SetDefault("security.enable_rbac_validation", false)
	v.SetDefault("security.enable_secret_masking", true)

	v.SetDefault("mcp.transport", "stdio")
	v.SetDefault("mcp.name", "sh-mcp-go")
	v.SetDefault("mcp.version", "1.0.0")
	v.SetDefault("mcp.sse_addr", "0.0.0.0:8081")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}
