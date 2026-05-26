package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func baseServiceYAML(logLevel string) string {
	return `
server:
  host: 127.0.0.1
  port: "9000"
postgres:
  host: postgres
  port: "5432"
  user: user
  password: pass
  db: app
redis:
  host: redis
  port: "6379"
  password: secret
  database: 2
app:
  shutdown_seconds: 5
  log_level: ` + logLevel + `
`
}

func TestCommonConfigHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "server info", got: (ServerConfig{Host: "127.0.0.1", Port: "8080"}).ServerInfo(), want: "127.0.0.1:8080"},
		{name: "postgres info", got: (PostgresConfig{Host: "db", Port: "5432"}).ServerInfo(), want: "db:5432"},
		{name: "redis info", got: (RedisConfig{Host: "redis", Port: "6379"}).ServerInfo(), want: "redis:6379"},
		{name: "s3 endpoint", got: (S3Config{Host: "s3", Port: "9000"}).Endpoint(), want: "s3:9000"},
		{name: "s3 endpoint url http", got: (S3Config{Host: "s3", Port: "9000"}).EndpointURL(), want: "http://s3:9000"},
		{name: "s3 endpoint url https", got: (S3Config{Host: "s3", Port: "9000", UseSSL: true}).EndpointURL(), want: "https://s3:9000"},
		{name: "s3 public url with port", got: (S3Config{PublicHost: "cdn", PublicPort: "9000", PublicPath: "/bucket"}).PublicURL(), want: "http://cdn:9000/bucket"},
		{name: "s3 public url without default port", got: (S3Config{PublicHost: "cdn", PublicPort: "443", PublicPath: "/bucket", PublicUseSSL: true}).PublicURL(), want: "https://cdn/bucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestParseZapLevel(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    zapcore.Level
		wantErr bool
	}{
		{name: "trim and lowercase", in: " DEBUG ", want: zapcore.DebugLevel},
		{name: "info", in: "info", want: zapcore.InfoLevel},
		{name: "invalid", in: "verbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseZapLevel(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseZapLevel() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseZapLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigPathFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		appConfig   string
		lowerConfig string
		want        string
	}{
		{name: "default", want: "configs/auth/local.yml"},
		{name: "config path wins", configPath: "/tmp/custom.yml", appConfig: "prod", want: "/tmp/custom.yml"},
		{name: "application config", appConfig: "prod", want: "configs/auth/prod.yml"},
		{name: "lowercase env fallback", lowerConfig: "stage", want: "configs/auth/stage.yml"},
		{name: "local uses default", appConfig: "local", want: "configs/auth/local.yml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", tt.configPath)
			t.Setenv("APPLICATION_CONFIG", tt.appConfig)
			t.Setenv("application_config", tt.lowerConfig)
			got, err := configPathFromEnv("configs/auth/local.yml")
			if err != nil {
				t.Fatalf("configPathFromEnv() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("configPathFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadAuthConfig(t *testing.T) {
	path := writeConfig(t, baseServiceYAML("debug")+`
session:
  ttl_seconds: 60
profile:
  grpc_addr: profile:8002
vkid:
  client_id: vk-client
  redirect_uri: https://example.com/callback
  auth_url: https://id.example.com/auth
  public_info_url: https://id.example.com/user
`)
	t.Setenv("CONFIG_PATH", path)

	cfg, err := LoadAuthConfig()
	if err != nil {
		t.Fatalf("LoadAuthConfig() error = %v", err)
	}
	if cfg.ServerConfig.ServerInfo() != "127.0.0.1:9000" {
		t.Fatalf("server = %+v", cfg.ServerConfig)
	}
	if cfg.SessionConfig.SessionTTL != time.Minute {
		t.Fatalf("ttl = %s", cfg.SessionConfig.SessionTTL)
	}
	if cfg.AuthProfileConfig.GRPCAddr != "profile:8002" || cfg.VKIDConfig.ClientID != "vk-client" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadChatConfigFromPath(t *testing.T) {
	path := writeConfig(t, baseServiceYAML("info")+`
ws_server:
  host: 0.0.0.0
  port: "9001"
media:
  grpc_addr: media:8003
profile:
  grpc_addr: profile:8002
subscription:
  grpc_addr: subscription:8011
profanity_roots_path: roots.txt
gateway_public_url: https://example.com
`)

	cfg, err := LoadChatConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadChatConfigFromPath() error = %v", err)
	}
	if cfg.WSServerConfig.ServerInfo() != "0.0.0.0:9001" || cfg.GatewayPublicURL != "https://example.com" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadMediaConfigFromPath(t *testing.T) {
	path := writeConfig(t, `
server:
  host: media
  port: "8003"
s3:
  host: minio
  port: "9000"
  bucket: bucket
  access_key: key
  secret_key: secret
  use_ssl: true
  public_use_ssl: true
  public_host: cdn.example.com
  public_port: "443"
  public_path: /media
speechkit:
  api_key: api
  lang: ru-RU
capybara:
  enabled: true
  score_threshold: 0.5
  python_path: python3
  timeout_seconds: 10
app:
  shutdown_seconds: 7
  log_level: warn
`)

	cfg, err := LoadMediaConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadMediaConfigFromPath() error = %v", err)
	}
	if cfg.S3Config.EndpointURL() != "https://minio:9000" || cfg.S3Config.PublicURL() != "https://cdn.example.com/media" {
		t.Fatalf("s3 = %+v", cfg.S3Config)
	}
	if cfg.CapybaraDetectorConfig.ScriptPath != "scripts/vision/detect_capybara.py" {
		t.Fatalf("script path = %q", cfg.CapybaraDetectorConfig.ScriptPath)
	}
}

func TestLoadProfileConfigFromPath(t *testing.T) {
	path := writeConfig(t, baseServiceYAML("info")+`
media:
  grpc_addr: media:8003
`)

	cfg, err := LoadProfileConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadProfileConfigFromPath() error = %v", err)
	}
	if cfg.ProfileMediaConfig.GRPCAddr != "media:8003" || cfg.RedisConfig.Database != 2 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadOtherServiceConfigs(t *testing.T) {
	tests := []struct {
		name string
		load func() (string, error)
		body string
	}{
		{
			name: "complaint",
			body: baseServiceYAML("info") + `
media:
  grpc_addr: media:8003
`,
			load: func() (string, error) {
				cfg, err := LoadComplaintConfig()
				if err != nil {
					return "", err
				}
				return cfg.ComplaintMediaConfig.GRPCAddr, nil
			},
		},
		{
			name: "search",
			body: baseServiceYAML("info"),
			load: func() (string, error) {
				cfg, err := LoadSearchConfig()
				if err != nil {
					return "", err
				}
				return cfg.RedisConfig.ServerInfo(), nil
			},
		},
		{
			name: "subscription",
			body: baseServiceYAML("info"),
			load: func() (string, error) {
				cfg, err := LoadSubscriptionConfig()
				if err != nil {
					return "", err
				}
				return cfg.PostgresConfig.Database, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CONFIG_PATH", writeConfig(t, tt.body))
			got, err := tt.load()
			if err != nil {
				t.Fatalf("load() error = %v", err)
			}
			if got == "" {
				t.Fatalf("load() returned empty marker")
			}
		})
	}
}

func TestLoadGatewayConfig(t *testing.T) {
	path := writeConfig(t, `
server:
  host: 127.0.0.1
  port: "8080"
public_base_url: https://example.com
session_cookie:
  secure: true
  http_only: true
  same_site: Strict
auth:
  grpc_addr: auth:8001
profile:
  grpc_addr: profile:8002
chat:
  ws_addr: chat:8005
  grpc_addr: chat:8004
media:
  grpc_addr: media:8003
complaint:
  grpc_addr: complaint:8006
search:
  grpc_addr: search:8010
subscription:
  grpc_addr: subscription:8011
payment:
  grpc_addr: payment:8012
`)

	cfg, err := LoadGatewayConfig(path)
	if err != nil {
		t.Fatalf("LoadGatewayConfig() error = %v", err)
	}
	if cfg.PublicBaseURL != "https://example.com" || cfg.Payment.GRPCAddr != "payment:8012" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadPaymentConfig(t *testing.T) {
	path := writeConfig(t, baseServiceYAML("info")+`
subscription:
  grpc_addr: subscription:8011
payment:
  secret_key: secret
  shop_id: shop
  return_url: https://example.com/payment/done
`)
	t.Setenv("CONFIG_PATH", path)

	cfg, err := LoadPaymentConfig()
	if err != nil {
		t.Fatalf("LoadPaymentConfig() error = %v", err)
	}
	if cfg.ReturnURL != "https://example.com/payment/done" || cfg.SubscriptionGRPC.GRPCAddr != "subscription:8011" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadGatewayConfig(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
			t.Fatalf("expected missing file error")
		}
	})

	t.Run("invalid log level", func(t *testing.T) {
		path := writeConfig(t, baseServiceYAML("verbose"))
		if _, err := LoadChatConfigFromPath(path); err == nil {
			t.Fatalf("expected invalid log level error")
		}
	})

	t.Run("invalid payment return url scheme", func(t *testing.T) {
		path := writeConfig(t, baseServiceYAML("info")+`
subscription:
  grpc_addr: subscription:8011
payment:
  return_url: ftp://example.com/payment/done
`)
		t.Setenv("CONFIG_PATH", path)
		if _, err := LoadPaymentConfig(); err == nil {
			t.Fatalf("expected invalid return url error")
		}
	})

	t.Run("invalid payment return url host", func(t *testing.T) {
		if err := validatePaymentReturnURL("https:///payment/done"); err == nil {
			t.Fatalf("expected invalid host error")
		}
	})
}
