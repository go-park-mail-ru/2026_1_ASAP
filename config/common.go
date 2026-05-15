package config

import (
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

type ServerConfig struct {
	Host string
	Port string
}

func (c ServerConfig) ServerInfo() string {
	return c.Host + ":" + c.Port
}

type PostgresConfig struct {
	Host              string
	Port              string
	Username          string
	Password          string
	Database          string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func (c PostgresConfig) ServerInfo() string {
	return c.Host + ":" + c.Port
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Database int
}

func (c RedisConfig) ServerInfo() string {
	return c.Host + ":" + c.Port
}

type SessionConfig struct {
	SessionTTL time.Duration
}

type AppConfig struct {
	ShutdownTime time.Duration
	LogLevel     zapcore.Level
}

type VKIDConfig struct {
	ClientID      string
	RedirectURI   string
	AuthURL       string
	PublicInfoURL string
}

type S3Config struct {
	Host         string
	Port         string
	Bucket       string
	AccessKey    string
	SecretKey    string
	PublicHost   string
	PublicPort   string
	PublicPath   string
	UseSSL       bool
	PublicUseSSL bool
}

func (c S3Config) Endpoint() string {
	return c.Host + ":" + c.Port
}

func (c S3Config) EndpointURL() string {
	scheme := "http"
	if c.UseSSL {
		scheme = "https"
	}
	return scheme + "://" + c.Host + ":" + c.Port
}

func (c S3Config) PublicURL() string {
	scheme := "http"
	if c.PublicUseSSL {
		scheme = "https"
	}
	var base string
	if c.PublicPort == "" || c.PublicPort == "80" || c.PublicPort == "443" {
		base = scheme + "://" + c.PublicHost
	} else {
		base = scheme + "://" + c.PublicHost + ":" + c.PublicPort
	}
	return base + c.PublicPath
}

func parseZapLevel(s string) (zapcore.Level, error) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(strings.ToLower(s)))); err != nil {
		return zapcore.InfoLevel, err
	}
	return l, nil
}
