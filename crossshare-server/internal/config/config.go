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
	Server    ServerConfig    `mapstructure:"server" yaml:"server"`
	Auth      AuthConfig      `mapstructure:"auth" yaml:"auth"`
	Business  BusinessConfig  `mapstructure:"business" yaml:"business"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit" yaml:"ratelimit"`
	CORS      CORSConfig      `mapstructure:"cors" yaml:"cors"`
	Storage   StorageConfig   `mapstructure:"storage" yaml:"storage"`
	Redis     RedisConfig     `mapstructure:"redis" yaml:"redis"`
}

type StorageConfig struct {
	Type string `mapstructure:"type" yaml:"type"`
}

type ServerConfig struct {
	Port      int    `mapstructure:"port" yaml:"port"`
	TLSEnable bool   `mapstructure:"tls_enable" yaml:"tls_enable"`
	CrtPath   string `mapstructure:"crt_path" yaml:"crt_path"`
	KeyPath   string `mapstructure:"key_path" yaml:"key_path"`
}

type AuthConfig struct {
	Enable        bool   `mapstructure:"enable" yaml:"enable"`
	JWTSecret     string `mapstructure:"jwt_secret" yaml:"jwt_secret"`
	JWTHeaderName string `mapstructure:"jwt_header_name" yaml:"jwt_header_name"`
}

type BusinessConfig struct {
	DefaultTTL      int   `mapstructure:"default_ttl" yaml:"default_ttl"`
	MaxTTL          int   `mapstructure:"max_ttl" yaml:"max_ttl"`
	TextJSONLimit   int64 `mapstructure:"text_json_limit" yaml:"text_json_limit"`
	BinaryPushLimit int64 `mapstructure:"binary_push_limit" yaml:"binary_push_limit"`
}

type RateLimitConfig struct {
	Enable            bool `mapstructure:"enable" yaml:"enable"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
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
	v.SetDefault("auth.enable", false)
	v.SetDefault("auth.jwt_secret", "change-me-in-production")
	v.SetDefault("auth.jwt_header_name", "Authorization")
	v.SetDefault("business.default_ttl", 600)
	v.SetDefault("business.max_ttl", 86400) // 24 hours
	v.SetDefault("business.text_json_limit", 1<<20)
	v.SetDefault("business.binary_push_limit", 20<<20)
	v.SetDefault("ratelimit.enable", true)
	v.SetDefault("ratelimit.requests_per_minute", 20)
	v.SetDefault("cors.allow_origins", []string{"*"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "DELETE", "OPTIONS"})
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
