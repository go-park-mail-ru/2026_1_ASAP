package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type MediaConfig struct {
	ServerConfig ServerConfig
	S3Config     S3Config
	AppConfig    AppConfig
}

type mediaFile struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
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

func LoadMediaConfig() (*MediaConfig, error) {
	path, err := configPathFromEnv("configs/media/local.yml")
	if err != nil {
		return nil, err
	}
	return LoadMediaConfigFromPath(path)
}

func LoadMediaConfigFromPath(path string) (*MediaConfig, error) {
	var raw mediaFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load media config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("media config log level %q: %w", raw.App.LogLevel, err)
	}

	return &MediaConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
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
