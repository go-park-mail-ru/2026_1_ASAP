package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type AuthProfileConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"auth:8001"`
}
type AuthConfig struct {
	PostgresConfig    PostgresConfig
	ServerConfig      ServerConfig
	RedisConfig       RedisConfig
	AuthProfileConfig AuthProfileConfig
	AppConfig         AppConfig
	SessionConfig     SessionConfig
	VKIDConfig        VKIDConfig
}

type authFile struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	Session struct {
		TTLSeconds int `yaml:"ttl_seconds"`
	} `yaml:"session"`
	Redis struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Database int    `yaml:"database"`
		Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	} `yaml:"redis"`
	AuthProfileConfig struct {
		GRPCAddr string `yaml:"grpc_addr" env-default:"auth:8001"`
	} `yaml:"profile"`
	Postgres struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		DB       string `yaml:"db"`
		Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:""`
	} `yaml:"postgres"`
	App struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
	VKID struct {
		ClientID      string `yaml:"client_id"`
		RedirectURI   string `yaml:"redirect_uri"`
		AuthURL       string `yaml:"auth_url"`
		PublicInfoURL string `yaml:"public_info_url"`
	} `yaml:"vkid"`
}

// LoadAuthConfig reads CONFIG_PATH (default configs/auth/local.yml) via cleanenv: YAML + env for secrets.
func LoadAuthConfig() (*AuthConfig, error) {
	path, err := configPathFromEnv("configs/auth/local.yml")
	if err != nil {
		return nil, err
	}

	var raw authFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load auth config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("auth config log level %q: %w", raw.App.LogLevel, err)
	}

	return &AuthConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		SessionConfig: SessionConfig{
			SessionTTL: time.Duration(raw.Session.TTLSeconds) * time.Second,
		},
		RedisConfig: RedisConfig{
			Host:     raw.Redis.Host,
			Port:     raw.Redis.Port,
			Password: raw.Redis.Password,
			Database: raw.Redis.Database,
		},
		AuthProfileConfig: AuthProfileConfig{
			GRPCAddr: raw.AuthProfileConfig.GRPCAddr,
		},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
		VKIDConfig: VKIDConfig{
			ClientID:      raw.VKID.ClientID,
			RedirectURI:   raw.VKID.RedirectURI,
			AuthURL:       raw.VKID.AuthURL,
			PublicInfoURL: raw.VKID.PublicInfoURL,
		},
	}, nil
}
