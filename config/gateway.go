package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type GatewayConfig struct {
	Server       GatewayServerConfig       `yaml:"server"`
	Auth         GatewayAuthConfig         `yaml:"auth"`
	Profile      GatewayProfileConfig      `yaml:"profile"`
	Chat         GatewayChatConfig         `yaml:"chat"`
	Complaint    GatewayComplaintConfig    `yaml:"complaint"`
	Search       GatewaySearchConfig       `yaml:"search"`
	Subscription GatewaySubscriptionConfig `yaml:"subscription"`
	Payment      GatewayPaymentConfig      `yaml:"payment"`
}

type GatewaySessionCookieConfig struct {
	Secure   bool   `yaml:"secure"`
	HTTPOnly bool   `yaml:"http_only"`
	SameSite string `yaml:"same_site" env-default:"Lax"`
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

type GatewayChatConfig struct {
	WSAddr   string `yaml:"ws_addr" env-default:"chat:8005"`
	GRPCAddr string `yaml:"grpc_addr" env-default:"chat:8004"`
}

type GatewayComplaintConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"complaint:8006"`
}

type GatewaySearchConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"search:8010"`
}

type GatewaySubscriptionConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"subscription:8011"`
}

type GatewayPaymentConfig struct {
	GRPCAddr string `yaml:"grpc_addr" env-default:"payment:8012"`
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
