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

func (postgresConfig PostgresConfig) ServerInfo() string {
	return postgresConfig.Host + ":" + postgresConfig.Port
}

func (redisConfig RedisConfig) ServerInfo() string {
	return redisConfig.Host + ":" + redisConfig.Port
}

func (serverConfig ServerConfig) ServerInfo() string {
	return serverConfig.Host + ":" + serverConfig.Port
}

func LoadConfigFromEnv() (*Config, error) {
	var serverConfig ServerConfig
	var sessionConfig SessionConfig
	var redisConfig RedisConfig
	var postgresConfig PostgresConfig

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

	passwordPostgres, err := getEnvVariable("POSTGRES_PASSWORD", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Password = passwordPostgres

	databasePostgres, err := getEnvVariable("POSTGRES_DB", "postgres")
	if err != nil {
		return nil, fmt.Errorf("Load Config error: %w", err)
	}
	postgresConfig.Database = databasePostgres

	config := &Config{
		ServerConfig:   serverConfig,
		SessionConfig:  sessionConfig,
		RedisConfig:    redisConfig,
		PostgresConfig: postgresConfig,
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
