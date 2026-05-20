package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ChatConfig struct {
	ServerConfig      ServerConfig
	WSServerConfig    ServerConfig
	PostgresConfig    PostgresConfig
	RedisConfig       RedisConfig
	ChatMediaConfig   ChatMediaConfig
	ChatProfileConfig ChatProfileConfig
	AppConfig         AppConfig
}

type ChatMediaConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"media:8003"`
}

type ChatProfileConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"profile:8002"`
}

type chatFile struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	WSServer struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"ws_server"`
	Postgres struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		DB       string `yaml:"db"`
		Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:""`
	} `yaml:"postgres"`
	Media   ChatMediaConfig   `yaml:"media"`
	Profile ChatProfileConfig `yaml:"profile"`
	Redis   struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
		Database int    `yaml:"database"`
	} `yaml:"redis"`
	App struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
}

func LoadChatConfig() (*ChatConfig, error) {
	path, err := configPathFromEnv("configs/chat/local.yml")
	if err != nil {
		return nil, err
	}
	return LoadChatConfigFromPath(path)
}

func LoadChatConfigFromPath(path string) (*ChatConfig, error) {
	var raw chatFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load chat config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("chat config log level %q: %w", raw.App.LogLevel, err)
	}

	return &ChatConfig{
		ServerConfig:   ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		WSServerConfig: ServerConfig{Host: raw.WSServer.Host, Port: raw.WSServer.Port},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
		ChatMediaConfig:   ChatMediaConfig{GRPCAddr: raw.Media.GRPCAddr},
		ChatProfileConfig: ChatProfileConfig{GRPCAddr: raw.Profile.GRPCAddr},
		RedisConfig: RedisConfig{
			Host:     raw.Redis.Host,
			Port:     raw.Redis.Port,
			Password: raw.Redis.Password,
			Database: raw.Redis.Database,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
