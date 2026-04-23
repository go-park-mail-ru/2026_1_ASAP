package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// ProfileConfig is the profile (gRPC) service configuration.
type ProfileConfig struct {
	ServerConfig   ServerConfig
	PostgresConfig PostgresConfig
	S3Config       S3Config
	AppConfig      AppConfig
}

type profileFile struct {
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
	S3 struct {
		Host         string `yaml:"host"`
		Port         string `yaml:"port"`
		Bucket       string `yaml:"bucket"`
		AccessKey    string `env:"S3_ACCESS_KEY" env-default:"minioadmin"`
		SecretKey    string `env:"S3_SECRET_KEY" env-default:"minioadmin"`
		UseSSL       bool   `yaml:"use_ssl"`
		PublicUseSSL bool   `yaml:"public_use_ssl"`
		PublicHost   string `yaml:"public_host"`
		PublicPort   string `yaml:"public_port"`
	} `yaml:"s3"`
	App struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
}

// LoadProfileConfig reads CONFIG_PATH (default configs/profile/local.yml) via cleanenv.
func LoadProfileConfig() (*ProfileConfig, error) {
	path, err := configPathFromEnv("configs/profile/local.yml")
	if err != nil {
		return nil, err
	}
	return LoadProfileConfigFromPath(path)
}

// LoadProfileConfigFromPath loads profile config from a given YAML path.
func LoadProfileConfigFromPath(path string) (*ProfileConfig, error) {
	var raw profileFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load profile config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("profile config log level %q: %w", raw.App.LogLevel, err)
	}

	return &ProfileConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
		S3Config: S3Config{
			Host:         raw.S3.Host,
			Port:         raw.S3.Port,
			Bucket:       raw.S3.Bucket,
			AccessKey:    raw.S3.AccessKey,
			SecretKey:    raw.S3.SecretKey,
			PublicHost:   raw.S3.PublicHost,
			PublicPort:   raw.S3.PublicPort,
			UseSSL:       raw.S3.UseSSL,
			PublicUseSSL: raw.S3.PublicUseSSL,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
