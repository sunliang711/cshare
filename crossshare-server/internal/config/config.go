package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

var Module = fx.Options(
	fx.Provide(New),
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server" yaml:"server"`
	CORS    CORSConfig    `mapstructure:"cors" yaml:"cors"`
	Storage StorageConfig `mapstructure:"storage" yaml:"storage"`
	Redis   RedisConfig   `mapstructure:"redis" yaml:"redis"`
}

type StorageConfig struct {
	Type string `mapstructure:"type" yaml:"type"`
}

type ServerConfig struct {
	Port      int    `mapstructure:"port" yaml:"port"`
	TLSEnable bool   `mapstructure:"tls_enable" yaml:"tls_enable"`
	CrtPath   string `mapstructure:"crt_path" yaml:"crt_path"`
	KeyPath   string `mapstructure:"key_path" yaml:"key_path"`
	DataDir   string `mapstructure:"data_dir" yaml:"data_dir"`
	MasterKey string `mapstructure:"master_key" yaml:"master_key"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins" yaml:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods" yaml:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers" yaml:"allow_headers"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr" yaml:"addr"`
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
	DB       int    `mapstructure:"db" yaml:"db"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 10431)
	v.SetDefault("server.data_dir", "./data")
	v.SetDefault("server.master_key", "")
	v.SetDefault("cors.allow_origins", []string{"*"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{
		"Content-Type", "Authorization", "X-Request-Id",
		"Filename", "X-Content-Type", "X-TTL",
		"Accept", "Delete-After-Pull",
	})
	v.SetDefault("storage.type", "memory")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.username", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
}

func New() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/crossshare")

	setDefaults(v)

	// Environment variable override: CS_SERVER_PORT, CS_REDIS_ADDR, etc.
	v.SetEnvPrefix("CS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		fmt.Println("[config] no config file found, using defaults and environment variables")
	} else {
		fmt.Printf("[config] using config file: %s\n", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Server.DataDir) == "" {
		cfg.Server.DataDir = "./data"
	}
	return &cfg, nil
}

// DefaultConfig returns the default configuration serialized as YAML bytes.
func DefaultConfig() ([]byte, error) {
	v := viper.New()
	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return yaml.Marshal(&cfg)
}
