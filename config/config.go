package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerConfig   ServerConfig
	SessionConfig  SessionConfig
	RedisConfig    RedisConfig
	PostgresConfig PostgresConfig
	S3Config       S3Config
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
	Host       string
	Port       string
	Bucket     string
	AccessKey  string
	SecretKey  string
	UseSSL     bool
	PublicHost string
	PublicPort string
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
	if c.UseSSL {
		scheme = "https"
	}
	if c.PublicPort == "" || c.PublicPort == "80" || c.PublicPort == "443" {
		return scheme + "://" + c.PublicHost
	}
	return scheme + "://" + c.PublicHost + ":" + c.PublicPort
}

func LoadConfigFromEnv() (*Config, error) {
	var serverConfig ServerConfig
	var sessionConfig SessionConfig
	var redisConfig RedisConfig
	var postgresConfig PostgresConfig
	var s3Config S3Config

	host, err := getEnvVariable("HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	serverConfig.Host = host

	port, err := getEnvVariable("PORT", "8080")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	serverConfig.Port = port

	sessionTTLString, err := getEnvVariable("TTL", "3600")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}

	sessionTTL, err := strconv.Atoi(sessionTTLString)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	sessionConfig.SessionTTL = time.Duration(sessionTTL) * time.Second

	hostRedis, err := getEnvVariable("REDIS_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Host = hostRedis

	portRedis, err := getEnvVariable("REDIS_PORT", "6379")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Port = portRedis

	passwordRedis, err := getEnvVariable("REDIS_PASSWORD", "")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Password = passwordRedis

	databaseRedisString, err := getEnvVariable("REDIS_DATABASE", "0")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}

	databaseRedis, err := strconv.Atoi(databaseRedisString)
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	redisConfig.Database = databaseRedis

	hostPostgres, err := getEnvVariable("POSTGRES_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Host = hostPostgres

	portPostgres, err := getEnvVariable("POSTGRES_PORT", "5432")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Port = portPostgres

	usernamePostgres, err := getEnvVariable("POSTGRES_USER", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Username = usernamePostgres

	passwordPostgres, err := getEnvVariable("POSTGRES_PASSWORD", "")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Password = passwordPostgres

	databasePostgres, err := getEnvVariable("POSTGRES_DB", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Database = databasePostgres

	s3Host, err := getEnvVariable("S3_HOST", "minio")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Host = s3Host

	s3Port, err := getEnvVariable("S3_PORT", "9000")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Port = s3Port

	s3AccessKey, err := getEnvVariable("S3_ACCESS_KEY", "minioadmin")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.AccessKey = s3AccessKey

	s3SecretKey, err := getEnvVariable("S3_SECRET_KEY", "minioadmin")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.SecretKey = s3SecretKey

	s3Bucket, err := getEnvVariable("S3_BUCKET", "media")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.Bucket = s3Bucket

	s3UseSSL, err := getEnvVariable("S3_USE_SSL", "false")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.UseSSL = s3UseSSL == "true" || s3UseSSL == "1"

	s3PublicHost, err := getEnvVariable("S3_PUBLIC_HOST", "localhost")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.PublicHost = s3PublicHost

	s3PublicPort, err := getEnvVariable("S3_PUBLIC_PORT", "9000")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	s3Config.PublicPort = s3PublicPort

	config := &Config{
		ServerConfig:   serverConfig,
		SessionConfig:  sessionConfig,
		RedisConfig:    redisConfig,
		PostgresConfig: postgresConfig,
		S3Config:       s3Config,
	}

	return config, nil
}

func getEnvVariable(variable string, defaultValue string) (string, error) {
	value, ok := os.LookupEnv(variable)
	if !ok {
		if defaultValue == "" {
			return "", fmt.Errorf("Variable %s not set (no default value provided)", variable)
		}
		log.Printf("Variable %s not found, using default %s", variable, defaultValue)
		return defaultValue, nil
	}
	log.Printf("Variable %s found, using %s", variable, value)
	return value, nil
}
