package config

import (
	"path/filepath"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

// configPathFromEnv returns path from CONFIG_PATH, or defaultPath when empty / unset.
func configPathFromEnv(defaultPath string) (string, error) {
	var env struct {
		ConfigPath         string `env:"CONFIG_PATH" env-default:""`
		ApplicationConfig  string `env:"APPLICATION_CONFIG" env-default:""`
		ApplicationConfig2 string `env:"application_config" env-default:""`
	}
	if err := cleanenv.ReadEnv(&env); err != nil {
		return "", err
	}
	if env.ConfigPath != "" {
		return env.ConfigPath, nil
	}
	appConfig := strings.TrimSpace(strings.ToLower(env.ApplicationConfig))
	if appConfig == "" {
		appConfig = strings.TrimSpace(strings.ToLower(env.ApplicationConfig2))
	}
	if appConfig != "" && appConfig != "local" {
		baseDir := filepath.Dir(defaultPath)
		return filepath.Join(baseDir, appConfig+".yml"), nil
	}
	return defaultPath, nil
}
