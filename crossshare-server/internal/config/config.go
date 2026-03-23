package config

import (
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(New),
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Business  BusinessConfig  `mapstructure:"business"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit"`
	CORS      CORSConfig      `mapstructure:"cors"`
	Redis     RedisConfig     `mapstructure:"redis"`
}

type ServerConfig struct {
	Port      int    `mapstructure:"port"`
	TLSEnable bool   `mapstructure:"tls_enable"`
	CrtPath   string `mapstructure:"crt_path"`
	KeyPath   string `mapstructure:"key_path"`
}

type AuthConfig struct {
	Enable        bool   `mapstructure:"enable"`
	JWTSecret     string `mapstructure:"jwt_secret"`
	JWTHeaderName string `mapstructure:"jwt_header_name"`
}

type BusinessConfig struct {
	DefaultTTL      int   `mapstructure:"default_ttl"`
	MaxTTL          int   `mapstructure:"max_ttl"`
	TextJSONLimit   int64 `mapstructure:"text_json_limit"`
	BinaryPushLimit int64 `mapstructure:"binary_push_limit"`
}

type RateLimitConfig struct {
	Enable            bool `mapstructure:"enable"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func New() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	v.SetDefault("server.port", 10431)
	v.SetDefault("auth.enable", false)
	v.SetDefault("auth.jwt_header_name", "Authorization")
	v.SetDefault("business.default_ttl", 600)
	v.SetDefault("business.max_ttl", 2592000)
	v.SetDefault("business.text_json_limit", 1<<20)
	v.SetDefault("business.binary_push_limit", 20<<20)
	v.SetDefault("ratelimit.enable", true)
	v.SetDefault("ratelimit.requests_per_minute", 60)
	v.SetDefault("cors.allow_origins", []string{"*"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{
		"Content-Type", "Authorization", "X-Request-Id",
		"Filename", "X-Content-Type", "X-TTL",
		"Accept", "Delete-After-Pull",
	})
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Environment variable override: CS_SERVER_PORT, CS_REDIS_ADDR, etc.
	v.SetEnvPrefix("CS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
