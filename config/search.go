package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type SearchConfig struct {
	ServerConfig   ServerConfig
	PostgresConfig PostgresConfig
	RedisConfig    RedisConfig
	AppConfig      AppConfig
}

type searchFile struct {
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
	Redis struct {
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

func LoadSearchConfig() (*SearchConfig, error) {
	path, err := configPathFromEnv("configs/search/local.yml")
	if err != nil {
		return nil, err
	}
	var raw searchFile
	if err = cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load search config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("search config log level %q: %w", raw.App.LogLevel, err)
	}

	return &SearchConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
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
