package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	ServerConfig   ServerConfig
	SessionConfig  SessionConfig
	RedisConfig    RedisConfig
	PostgresConfig PostgresConfig
	S3Config       S3Config
	AppConfig      AppConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type SessionConfig struct {
	SessionTTL time.Duration
}
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Database int
}

type PostgresConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type S3Config struct {
	Host         string
	Port         string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UseSSL       bool
	PublicUseSSL bool
	PublicHost   string
	PublicPort   string
}

type AppConfig struct {
	ShutdownTime time.Duration
	LogLevel     zapcore.Level
}

func parseZapLevel(s string) (zapcore.Level, error) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(strings.ToLower(s)))); err != nil {
		return zapcore.InfoLevel, err
	}
	return l, nil
}

func (postgresConfig PostgresConfig) ServerInfo() string {
	return postgresConfig.Host + ":" + postgresConfig.Port
}

func (redisConfig RedisConfig) ServerInfo() string {
	return redisConfig.Host + ":" + redisConfig.Port
}

func (serverConfig ServerConfig) ServerInfo() string {
	return serverConfig.Host + ":" + serverConfig.Port
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

func LoadConfig() (*Config, error) {
	loadLogger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("zap bootstrap: %w", err)
	}
	defer func() { _ = loadLogger.Sync() }()
	return LoadConfigFromEnv(loadLogger)
}

func LoadConfigFromEnv(logger *zap.Logger) (*Config, error) {
	if logger == nil {
		log.Fatalln("logger is nil")
	}

	var serverConfig ServerConfig
	var sessionConfig SessionConfig
	var redisConfig RedisConfig
	var postgresConfig PostgresConfig
	var s3Config S3Config
	var appConfig AppConfig

	host, err := getEnvVariable(logger, "HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	serverConfig.Host = host

	port, err := getEnvVariable(logger, "PORT", "8080")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	serverConfig.Port = port

	sessionTTLString, err := getEnvVariable(logger, "TTL", "3600")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}

	sessionTTL, err := strconv.Atoi(sessionTTLString)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	sessionConfig.SessionTTL = time.Duration(sessionTTL) * time.Second

	hostRedis, err := getEnvVariable(logger, "REDIS_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Host = hostRedis

	portRedis, err := getEnvVariable(logger, "REDIS_PORT", "6379")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Port = portRedis

	passwordRedis, err := getEnvVariable(logger, "REDIS_PASSWORD", "")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Password = passwordRedis

	databaseRedisString, err := getEnvVariable(logger, "REDIS_DATABASE", "0")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}

	databaseRedis, err := strconv.Atoi(databaseRedisString)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Database = databaseRedis

	hostPostgres, err := getEnvVariable(logger, "POSTGRES_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Host = hostPostgres

	portPostgres, err := getEnvVariable(logger, "POSTGRES_PORT", "5432")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Port = portPostgres

	usernamePostgres, err := getEnvVariable(logger, "POSTGRES_USER", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Username = usernamePostgres

	passwordPostgres, err := getEnvVariable(logger, "POSTGRES_PASSWORD", "")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Password = passwordPostgres

	databasePostgres, err := getEnvVariable(logger, "POSTGRES_DB", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Database = databasePostgres

	s3Host, err := getEnvVariable(logger, "S3_HOST", "minio")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Host = s3Host

	s3Port, err := getEnvVariable(logger, "S3_PORT", "9000")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Port = s3Port

	s3AccessKey, err := getEnvVariable(logger, "S3_ACCESS_KEY", "minioadmin")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.AccessKey = s3AccessKey

	s3SecretKey, err := getEnvVariable(logger, "S3_SECRET_KEY", "minioadmin")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.SecretKey = s3SecretKey

	s3Bucket, err := getEnvVariable(logger, "S3_BUCKET", "media")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Bucket = s3Bucket

	s3UseSSL, err := getEnvVariable(logger, "S3_USE_SSL", "false")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.UseSSL = s3UseSSL == "true" || s3UseSSL == "1"

	s3PublicUseSSLStr, err := getEnvVariable(logger, "S3_PUBLIC_USE_SSL", "true")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.PublicUseSSL = s3PublicUseSSLStr == "true" || s3PublicUseSSLStr == "1"

	s3PublicHost, err := getEnvVariable(logger, "S3_PUBLIC_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.PublicHost = s3PublicHost

	s3PublicPort, err := getEnvVariable(logger, "S3_PUBLIC_PORT", "9000")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.PublicPort = s3PublicPort

	shutdownTimeString, err := getEnvVariable(logger, "SHUTDOWN_TIME", "10")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}

	ShutdownTime, err := strconv.Atoi(shutdownTimeString)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	appConfig.ShutdownTime = time.Duration(ShutdownTime) * time.Second

	logLevelStr, err := getEnvVariable(logger, "LOG_LEVEL", "info")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	logLevel, err := parseZapLevel(logLevelStr)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: LOG_LEVEL=%q: %w", logLevelStr, err)
	}
	appConfig.LogLevel = logLevel

	config := &Config{
		ServerConfig:   serverConfig,
		SessionConfig:  sessionConfig,
		RedisConfig:    redisConfig,
		PostgresConfig: postgresConfig,
		S3Config:       s3Config,
		AppConfig:      appConfig,
	}

	logger.Info("config loaded",
		zap.String("http_listen", serverConfig.ServerInfo()),
		zap.String("postgres", postgresConfig.ServerInfo()),
		zap.String("redis", redisConfig.ServerInfo()),
		zap.String("s3_endpoint", s3Config.Endpoint()),
		zap.String("s3_bucket", s3Config.Bucket),
		zap.Bool("s3_use_ssl", s3Config.UseSSL),
		zap.Bool("s3_public_use_ssl", s3Config.PublicUseSSL),
		zap.String("s3_public_url", s3Config.PublicURL()),
		zap.Duration("shutdown_timeout", appConfig.ShutdownTime),
		zap.String("log_level", appConfig.LogLevel.String()),
	)

	return config, nil
}

func redactEnvKey(key string) bool {
	u := strings.ToUpper(key)
	return strings.Contains(u, "PASSWORD") || strings.Contains(u, "SECRET") || strings.HasSuffix(u, "_KEY")
}

func envValueForLog(key, value string) string {
	if redactEnvKey(key) {
		if value == "" {
			return "(empty)"
		}
		return "***"
	}
	return value
}

func getEnvVariable(logger *zap.Logger, variable string, defaultValue string) (string, error) {
	value, ok := os.LookupEnv(variable)
	if !ok {
		if defaultValue == "" {
			return "", fmt.Errorf("Variable %s not set (no default value provided)", variable)
		}
		logger.Debug(".env: using default",
			zap.String("var", variable),
			zap.String("value", envValueForLog(variable, defaultValue)),
		)
		return defaultValue, nil
	}
	logger.Debug(".env: set",
		zap.String("var", variable),
		zap.String("value", envValueForLog(variable, value)),
	)
	return value, nil
}
