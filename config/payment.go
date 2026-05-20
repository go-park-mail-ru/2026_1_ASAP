package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type PaymentConfig struct {
	ServerConfig     ServerConfig
	PostgresConfig   PostgresConfig
	AppConfig        AppConfig
	SubscriptionGRPC SubscriptionGRPCConfig
	SecretKey        string
	ShopID           string
	ReturnURL        string
}

type SubscriptionGRPCConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
}

type paymentFile struct {
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
	Subscription struct {
		GRPCAddr string `yaml:"grpc_addr" env-default:"subscription:8011"`
	} `yaml:"subscription"`
	Payment struct {
		SecretKey string `yaml:"secret_key" env:"PAYMENT_SECRET_KEY" env-default:""`
		ShopID    string `yaml:"shop_id" env:"PAYMENT_SHOP_ID" env-default:""`
		ReturnURL string `yaml:"return_url" env:"PAYMENT_RETURN_URL" env-default:""`
	} `yaml:"payment"`
	App struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
}

func LoadPaymentConfig() (*PaymentConfig, error) {
	path, err := configPathFromEnv("configs/payment/local.yml")
	if err != nil {
		return nil, err
	}
	var raw paymentFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load payment config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("payment config log level %q: %w", raw.App.LogLevel, err)
	}

	returnURL := strings.TrimSpace(raw.Payment.ReturnURL)
	if returnURL != "" {
		if err := validatePaymentReturnURL(returnURL); err != nil {
			return nil, err
		}
	}

	return &PaymentConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		PostgresConfig: PostgresConfig{
			Host:     raw.Postgres.Host,
			Port:     raw.Postgres.Port,
			Username: raw.Postgres.User,
			Password: raw.Postgres.Password,
			Database: raw.Postgres.DB,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
		SubscriptionGRPC: SubscriptionGRPCConfig{
			GRPCAddr: raw.Subscription.GRPCAddr,
		},
		SecretKey: raw.Payment.SecretKey,
		ShopID:    raw.Payment.ShopID,
		ReturnURL: returnURL,
	}, nil
}

func validatePaymentReturnURL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("payment.return_url: parse %q: %w", s, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("payment.return_url must be an absolute URL with http or https scheme (e.g. https://pulseapp.space/payment/done), got %q", s)
	}
	if u.Host == "" {
		return fmt.Errorf("payment.return_url must include host (e.g. https://pulseapp.space/...), got %q", s)
	}
	return nil
}
