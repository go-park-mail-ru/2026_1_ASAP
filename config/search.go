package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type SearchConfig struct {
	ServerConfig   ServerConfig
	PostgresConfig PostgresConfig
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
		Password string `yaml:"password" env:"ASAP_APP_DB_PASSWORD" env-default:""`
		MaxConns int32  `yaml:"max_conns" env-default:"10"`
		MinConns int32  `yaml:"min_conns" env-default:"2"`
		MaxConnLifetimeSeconds int `yaml:"max_conn_lifetime_seconds" env-default:"1800"`
		MaxConnIdleTimeSeconds int `yaml:"max_conn_idle_time_seconds" env-default:"300"`
		HealthCheckPeriodSeconds int `yaml:"health_check_period_seconds" env-default:"30"`
	} `yaml:"postgres"`
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
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
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
			MaxConns: raw.Postgres.MaxConns,
			MinConns: raw.Postgres.MinConns,
			MaxConnLifetime: time.Duration(raw.Postgres.MaxConnLifetimeSeconds) * time.Second,
			MaxConnIdleTime: time.Duration(raw.Postgres.MaxConnIdleTimeSeconds) * time.Second,
			HealthCheckPeriod: time.Duration(raw.Postgres.HealthCheckPeriodSeconds) * time.Second,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
