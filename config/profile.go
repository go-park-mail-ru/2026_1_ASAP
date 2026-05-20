package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// ProfileConfig is the profile (gRPC) service configuration.
type ProfileConfig struct {
	ServerConfig       ServerConfig
	PostgresConfig     PostgresConfig
	RedisConfig        RedisConfig
	ProfileMediaConfig ProfileMediaConfig
	S3Config           S3Config
	AppConfig          AppConfig
}

type ProfileMediaConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"auth:8001"`
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
	ProfileMediaConfig ProfileMediaConfig `yaml:"media"`
	Redis              struct {
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
		ProfileMediaConfig: ProfileMediaConfig{
			GRPCAddr: raw.ProfileMediaConfig.GRPCAddr,
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
