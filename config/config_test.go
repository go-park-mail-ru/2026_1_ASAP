package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestPositiveConfig_ParseZapLevel(t *testing.T) {
	type fields struct{}

	type args struct {
		input string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    zapcore.Level
	}{
		{name: "info", prepare: nil, args: args{input: "info"}, want: zapcore.InfoLevel},
		{name: "INFO trimmed", prepare: nil, args: args{input: "  INFO  "}, want: zapcore.InfoLevel},
		{name: "debug", prepare: nil, args: args{input: "debug"}, want: zapcore.DebugLevel},
		{name: "error", prepare: nil, args: args{input: "error"}, want: zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got, err := parseZapLevel(tt.args.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeConfig_ParseZapLevel(t *testing.T) {
	type fields struct{}

	type args struct {
		input string
	}

	tests := []struct {
		prepare    func(*fields)
		name       string
		args       args
		wantSubstr string
		wantErr    bool
		wantLevel  zapcore.Level
	}{
		{
			name:       "unknown level",
			prepare:    nil,
			args:       args{input: "not-a-real-level-xyz"},
			wantErr:    true,
			wantLevel:  zapcore.InfoLevel,
			wantSubstr: "unrecognized level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got, err := parseZapLevel(tt.args.input)
			require.Equal(t, tt.wantLevel, got)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
}

func TestPositiveConfig_RedactEnvKey(t *testing.T) {
	type fields struct{}

	type args struct {
		key string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    bool
	}{
		{name: "password substring", prepare: nil, args: args{key: "POSTGRES_PASSWORD"}, want: true},
		{name: "secret substring", prepare: nil, args: args{key: "S3_SECRET_KEY"}, want: true},
		{name: "_KEY suffix", prepare: nil, args: args{key: "API_KEY"}, want: true},
		{name: "host not redacted", prepare: nil, args: args{key: "HOST"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			require.Equal(t, tt.want, redactEnvKey(tt.args.key))
		})
	}
}

func TestPositiveConfig_EnvValueForLog(t *testing.T) {
	type fields struct{}

	type args struct {
		key   string
		value string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    string
	}{
		{name: "redacted non-empty", prepare: nil, args: args{key: "SECRET", value: "hunter2"}, want: "***"},
		{name: "redacted empty", prepare: nil, args: args{key: "PASSWORD", value: ""}, want: "(empty)"},
		{name: "plain value", prepare: nil, args: args{key: "HOST", value: "127.0.0.1"}, want: "127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			require.Equal(t, tt.want, envValueForLog(tt.args.key, tt.args.value))
		})
	}
}

func TestPositiveConfig_ServerInfo(t *testing.T) {
	type fields struct{}

	type args struct {
		postgres PostgresConfig
		server   ServerConfig
		redis    RedisConfig
	}

	tests := []struct {
		prepare    func(*fields)
		name       string
		wantServer string
		wantRedis  string
		wantPG     string
		args       args
	}{
		{
			name:    "concatenates host and port",
			prepare: nil,
			args: args{
				server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				redis:    RedisConfig{Host: "redis", Port: "6379"},
				postgres: PostgresConfig{Host: "db", Port: "5432"},
			},
			wantServer: "0.0.0.0:8080",
			wantRedis:  "redis:6379",
			wantPG:     "db:5432",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			require.Equal(t, tt.wantServer, tt.args.server.ServerInfo())
			require.Equal(t, tt.wantRedis, tt.args.redis.ServerInfo())
			require.Equal(t, tt.wantPG, tt.args.postgres.ServerInfo())
		})
	}
}

func TestPositiveConfig_S3ConfigURLs(t *testing.T) {
	type fields struct{}

	type args struct {
		endpoint string
		url      string
		public   string
		cfg      S3Config
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name:    "internal http and public https default port",
			prepare: nil,
			args: args{
				cfg: S3Config{
					Host: "minio", Port: "9000", UseSSL: false,
					PublicHost: "cdn.example.com", PublicPort: "443", PublicUseSSL: true,
				},
				endpoint: "minio:9000",
				url:      "http://minio:9000",
				public:   "https://cdn.example.com",
			},
		},
		{
			name:    "internal https and public host with explicit port",
			prepare: nil,
			args: args{
				cfg: S3Config{
					Host: "s3", Port: "443", UseSSL: true,
					PublicHost: "files.app", PublicPort: "9001", PublicUseSSL: false,
				},
				endpoint: "s3:443",
				url:      "https://s3:443",
				public:   "http://files.app:9001",
			},
		},
		{
			name:    "public port 80 omitted",
			prepare: nil,
			args: args{
				cfg: S3Config{
					Host: "a", Port: "1", UseSSL: false,
					PublicHost: "x.test", PublicPort: "80", PublicUseSSL: true,
				},
				endpoint: "a:1",
				url:      "http://a:1",
				public:   "https://x.test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			require.Equal(t, tt.args.endpoint, tt.args.cfg.Endpoint())
			require.Equal(t, tt.args.url, tt.args.cfg.EndpointURL())
			require.Equal(t, tt.args.public, tt.args.cfg.PublicURL())
		})
	}
}

func TestPositiveConfig_LoadConfigFromEnv(t *testing.T) {
	type fields struct{}

	type args struct{}

	tests := []struct {
		args    args
		prepare func(*fields, *testing.T)
		assert  func(*testing.T, *Config)
		name    string
	}{
		{
			name: "Defaults when optional secrets empty in env",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
			},
			args: args{},
			assert: func(t *testing.T, cfg *Config) {
				require.Equal(t, "localhost", cfg.ServerConfig.Host)
				require.Equal(t, "8080", cfg.ServerConfig.Port)
				require.Equal(t, 3600*time.Second, cfg.SessionConfig.SessionTTL)
				require.Equal(t, zapcore.InfoLevel, cfg.AppConfig.LogLevel)
				require.Equal(t, 10*time.Second, cfg.AppConfig.ShutdownTime)
				require.Equal(t, "minio", cfg.S3Config.Host)
				require.False(t, cfg.S3Config.UseSSL)
			},
		},
		{
			name: "Overrides from env",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
				t.Setenv("HOST", "api.example")
				t.Setenv("PORT", "3000")
				t.Setenv("TTL", "120")
				t.Setenv("LOG_LEVEL", "warn")
				t.Setenv("S3_USE_SSL", "true")
			},
			args: args{},
			assert: func(t *testing.T, cfg *Config) {
				require.Equal(t, "api.example", cfg.ServerConfig.Host)
				require.Equal(t, "3000", cfg.ServerConfig.Port)
				require.Equal(t, 120*time.Second, cfg.SessionConfig.SessionTTL)
				require.Equal(t, zapcore.WarnLevel, cfg.AppConfig.LogLevel)
				require.True(t, cfg.S3Config.UseSSL)
			},
		},
	}

	logger := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f, t)
			}
			cfg, err := LoadConfigFromEnv(logger)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			tt.assert(t, cfg)
		})
	}
}

func TestNegativeConfig_LoadConfigFromEnv(t *testing.T) {
	type fields struct{}

	type args struct{}

	tests := []struct {
		name       string
		prepare    func(*fields, *testing.T)
		args       args
		wantSubstr string
	}{
		{
			name: "Invalid TTL",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
				t.Setenv("TTL", "nan")
			},
			wantSubstr: "Load Config error",
		},
		{
			name: "Invalid REDIS_DATABASE",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
				t.Setenv("REDIS_DATABASE", "x")
			},
			wantSubstr: "Load Config error",
		},
		{
			name: "Invalid SHUTDOWN_TIME",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
				t.Setenv("SHUTDOWN_TIME", "oops")
			},
			wantSubstr: "Load Config error",
		},
		{
			name: "Invalid LOG_LEVEL",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("REDIS_PASSWORD", "")
				t.Setenv("POSTGRES_PASSWORD", "")
				t.Setenv("LOG_LEVEL", "superman")
			},
			wantSubstr: "LOG_LEVEL",
		},
	}

	logger := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f, t)
			}
			cfg, err := LoadConfigFromEnv(logger)
			require.Nil(t, cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
}

func TestPositiveConfig_GetEnvVariable(t *testing.T) {
	type fields struct{}

	type args struct {
		key   string
		def   string
		unset bool
	}

	tests := []struct {
		prepare func(*fields, *testing.T)
		name    string
		want    string
		args    args
	}{
		{
			name:    "Uses default when unset",
			prepare: nil,
			args:    args{key: "CONFIG_TEST_GETENV_XYZ", def: "default-val", unset: true},
			want:    "default-val",
		},
		{
			name: "Uses env when set",
			prepare: func(_ *fields, t *testing.T) {
				t.Setenv("CONFIG_TEST_GETENV_ABC", "from-env")
			},
			args: args{key: "CONFIG_TEST_GETENV_ABC", def: "ignored", unset: false},
			want: "from-env",
		},
	}

	logger := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f, t)
			}
			if tt.args.unset {
				_ = os.Unsetenv(tt.args.key)
			}
			got, err := getEnvVariable(logger, tt.args.key, tt.args.def)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeConfig_GetEnvVariable(t *testing.T) {
	type fields struct{}

	type args struct {
		key string
	}

	tests := []struct {
		name    string
		prepare func(*fields, *testing.T)
		args    args
	}{
		{
			name: "Unset and no default",
			prepare: func(_ *fields, _ *testing.T) {
				_ = os.Unsetenv("CONFIG_TEST_GETENV_UNSET_NO_DEFAULT")
			},
			args: args{key: "CONFIG_TEST_GETENV_UNSET_NO_DEFAULT"},
		},
	}

	logger := zap.NewNop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f, t)
			}
			_, err := getEnvVariable(logger, tt.args.key, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), "not set")
		})
	}
}
