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
		Password string `yaml:"password" env:"ASAP_APP_DB_PASSWORD" env-default:""`
		MaxConns int32  `yaml:"max_conns" env-default:"10"`
		MinConns int32  `yaml:"min_conns" env-default:"2"`
		MaxConnLifetimeSeconds int `yaml:"max_conn_lifetime_seconds" env-default:"1800"`
		MaxConnIdleTimeSeconds int `yaml:"max_conn_idle_time_seconds" env-default:"300"`
		HealthCheckPeriodSeconds int `yaml:"health_check_period_seconds" env-default:"30"`
	} `yaml:"postgres"`
	Media   ChatMediaConfig   `yaml:"media"`
	Profile ChatProfileConfig `yaml:"profile"`
	App     struct {
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
			MaxConns: raw.Postgres.MaxConns,
			MinConns: raw.Postgres.MinConns,
			MaxConnLifetime: time.Duration(raw.Postgres.MaxConnLifetimeSeconds) * time.Second,
			MaxConnIdleTime: time.Duration(raw.Postgres.MaxConnIdleTimeSeconds) * time.Second,
			HealthCheckPeriod: time.Duration(raw.Postgres.HealthCheckPeriodSeconds) * time.Second,
		},
		ChatMediaConfig:   ChatMediaConfig{GRPCAddr: raw.Media.GRPCAddr},
		ChatProfileConfig: ChatProfileConfig{GRPCAddr: raw.Profile.GRPCAddr},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
