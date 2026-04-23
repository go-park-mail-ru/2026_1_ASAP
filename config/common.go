package config

import (
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// --- Shared blocks used by several services and repositories.

type ServerConfig struct {
	Host string
	Port string
}

func (c ServerConfig) ServerInfo() string {
	return c.Host + ":" + c.Port
}

type PostgresConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
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
	if c.PublicPort == "" || c.PublicPort == "80" || c.PublicPort == "443" {
		return scheme + "://" + c.PublicHost
	}
	return scheme + "://" + c.PublicHost + ":" + c.PublicPort
}

func parseZapLevel(s string) (zapcore.Level, error) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(strings.ToLower(s)))); err != nil {
		return zapcore.InfoLevel, err
	}
	return l, nil
}
