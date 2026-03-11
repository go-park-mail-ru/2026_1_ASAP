package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerConfig  ServerConfig
	SessionConfig SessionConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type SessionConfig struct {
	SessionTTL time.Duration
}

func (serverConfig ServerConfig) ServerInfo() string {
	return serverConfig.Host + ":" + serverConfig.Port
}

func LoadConfigFromEnv() (*Config, error) {
	var serverConfig ServerConfig
	var sessionConfig SessionConfig

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
	config := &Config{
		ServerConfig:  serverConfig,
		SessionConfig: sessionConfig,
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

	return value, nil
}
