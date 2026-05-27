package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ComplaintConfig struct {
	ServerConfig         ServerConfig
	PostgresConfig       PostgresConfig
	ComplaintMediaConfig ComplaintMediaConfig
	AppConfig            AppConfig
}

type ComplaintMediaConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"media:8003"`
}

type complaintFile struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	Postgres struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		DB       string `yaml:"db"`
		Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:""`
	} `yaml:"postgres"`
	Media ComplaintMediaConfig `yaml:"media"`
	App   struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
}

func LoadComplaintConfig() (*ComplaintConfig, error) {
	path, err := configPathFromEnv("configs/complaint/local.yml")
	if err != nil {
		return nil, err
	}

	var raw complaintFile
	if err = cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load complaint config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("complaint config log level %q: %w", raw.App.LogLevel, err)
	}

	return &ComplaintConfig{
		ServerConfig: ServerConfig{
			Host: raw.Server.Host,
			Port: raw.Server.Port,
		},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
		ComplaintMediaConfig: ComplaintMediaConfig{
			GRPCAddr: raw.Media.GRPCAddr,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
