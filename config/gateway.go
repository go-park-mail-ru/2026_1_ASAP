package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type GatewayConfig struct {
	Server  GatewayServerConfig  `yaml:"server"`
	Auth    GatewayAuthConfig    `yaml:"auth"`
	Profile GatewayProfileConfig `yaml:"profile"`
}

type GatewayServerConfig struct {
	Host string `yaml:"host" env-default:"0.0.0.0"`
	Port string `yaml:"port" env-default:"8080"`
}

type GatewayAuthConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"auth:8001"`
}

type GatewayProfileConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"profile:8002"`
}

func LoadGatewayConfig(path string) (*GatewayConfig, error) {
	if path == "" {
		var err error
		path, err = configPathFromEnv("configs/gateway/local.yml")
		if err != nil {
			return nil, err
		}
	}

	var cfg GatewayConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("load gateway config %q: %w", path, err)
	}
	return &cfg, nil
}
