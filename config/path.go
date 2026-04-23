package config

import "github.com/ilyakaznacheev/cleanenv"

// configPathFromEnv returns path from CONFIG_PATH, or defaultPath when empty / unset.
func configPathFromEnv(defaultPath string) (string, error) {
	var env struct {
		ConfigPath string `env:"CONFIG_PATH" env-default:""`
	}
	if err := cleanenv.ReadEnv(&env); err != nil {
		return "", err
	}
	if env.ConfigPath != "" {
		return env.ConfigPath, nil
	}
	return defaultPath, nil
}
