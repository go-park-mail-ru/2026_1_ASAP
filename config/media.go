package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type MediaConfig struct {
	ServerConfig           ServerConfig
	S3Config               S3Config
	SpeechKitConfig        SpeechKitConfig
	CapybaraDetectorConfig CapybaraDetectorConfig
	AppConfig              AppConfig
}

type CapybaraDetectorConfig struct {
	Enabled        bool    `yaml:"enabled" env:"CAPYBARA_DETECTOR_ENABLED" env-default:"true"`
	ScoreThreshold float64 `yaml:"score_threshold" env:"CAPYBARA_SCORE_THRESHOLD" env-default:"0.28"`
	PythonPath     string  `yaml:"python_path" env:"CAPYBARA_PYTHON" env-default:"python3"`
	ScriptPath     string  `yaml:"script_path" env-default:"scripts/vision/detect_capybara.py"`
	TimeoutSeconds int     `yaml:"timeout_seconds" env-default:"60"`
}

type SpeechKitConfig struct {
	APIKey string `yaml:"api_key" env:"SPEECHKIT_API_KEY" env-default:""`
	Lang   string `yaml:"lang" env-default:"ru-RU"`
}

type mediaFile struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	S3 struct {
		Host         string `yaml:"host"`
		Port         string `yaml:"port"`
		Bucket       string `yaml:"bucket"`
		AccessKey    string `env:"S3_ACCESS_KEY" env-default:"minioadmin"`
		SecretKey    string `env:"S3_SECRET_KEY" env-default:"minioadmin"`
		UseSSL       bool   `yaml:"use_ssl"`
		PublicUseSSL bool   `yaml:"public_use_ssl"`
		PublicHost   string `yaml:"public_host"`
		PublicPort   string `yaml:"public_port"`
		PublicPath   string `yaml:"public_path"`
	} `yaml:"s3"`
	SpeechKit struct {
		APIKey string `yaml:"api_key" env:"SPEECHKIT_API_KEY" env-default:""`
		Lang   string `yaml:"lang" env-default:"ru-RU"`
	} `yaml:"speechkit"`
	Capybara struct {
		Enabled        bool    `yaml:"enabled" env:"CAPYBARA_DETECTOR_ENABLED" env-default:"true"`
		ScoreThreshold float64 `yaml:"score_threshold" env:"CAPYBARA_SCORE_THRESHOLD" env-default:"0.28"`
		PythonPath     string  `yaml:"python_path" env:"CAPYBARA_PYTHON" env-default:"python3"`
		ScriptPath     string  `yaml:"script_path" env-default:"scripts/vision/detect_capybara.py"`
		TimeoutSeconds int     `yaml:"timeout_seconds" env-default:"60"`
	} `yaml:"capybara"`
	App struct {
		ShutdownSeconds int    `yaml:"shutdown_seconds"`
		LogLevel        string `yaml:"log_level"`
	} `yaml:"app"`
}

func LoadMediaConfig() (*MediaConfig, error) {
	path, err := configPathFromEnv("configs/media/local.yml")
	if err != nil {
		return nil, err
	}
	return LoadMediaConfigFromPath(path)
}

func LoadMediaConfigFromPath(path string) (*MediaConfig, error) {
	var raw mediaFile
	if err := cleanenv.ReadConfig(path, &raw); err != nil {
		return nil, fmt.Errorf("load media config %q: %w", path, err)
	}

	logLevel, err := parseZapLevel(raw.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("media config log level %q: %w", raw.App.LogLevel, err)
	}

	scriptPath := raw.Capybara.ScriptPath
	if scriptPath == "" {
		scriptPath = "scripts/vision/detect_capybara.py"
	}
	return &MediaConfig{
		ServerConfig: ServerConfig{Host: raw.Server.Host, Port: raw.Server.Port},
		SpeechKitConfig: SpeechKitConfig{
			APIKey: raw.SpeechKit.APIKey,
			Lang:   raw.SpeechKit.Lang,
		},
		CapybaraDetectorConfig: CapybaraDetectorConfig{
			Enabled:        raw.Capybara.Enabled,
			ScoreThreshold: raw.Capybara.ScoreThreshold,
			PythonPath:     raw.Capybara.PythonPath,
			ScriptPath:     scriptPath,
			TimeoutSeconds: raw.Capybara.TimeoutSeconds,
		},
		S3Config: S3Config{
			Host:         raw.S3.Host,
			Port:         raw.S3.Port,
			Bucket:       raw.S3.Bucket,
			AccessKey:    raw.S3.AccessKey,
			SecretKey:    raw.S3.SecretKey,
			PublicHost:   raw.S3.PublicHost,
			PublicPort:   raw.S3.PublicPort,
			PublicPath:   raw.S3.PublicPath,
			UseSSL:       raw.S3.UseSSL,
			PublicUseSSL: raw.S3.PublicUseSSL,
		},
		AppConfig: AppConfig{
			ShutdownTime: time.Duration(raw.App.ShutdownSeconds) * time.Second,
			LogLevel:     logLevel,
		},
	}, nil
}
