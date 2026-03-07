package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsServerConfig_Validate(t *testing.T) {
	baseCfg := Config{
		DBDriver:                  "postgres",
		DBConnectionString:        "postgres://localhost",
		ServerPort:                8080,
		MetricsPort:               8081,
		LogLevel:                  "info",
		ServerReadTimeout:         15 * time.Second,
		ServerWriteTimeout:        15 * time.Second,
		ServerIdleTimeout:         60 * time.Second,
		MaxRequestBodySize:        1048576,
		SecretValueSizeLimitBytes: 524288,
		MetricsServerReadTimeout:  15 * time.Second,
		MetricsServerWriteTimeout: 15 * time.Second,
		MetricsServerIdleTimeout:  60 * time.Second,
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid metrics server timeouts",
			cfg:     baseCfg,
			wantErr: false,
		},
		{
			name: "invalid metrics server read timeout - below minimum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerReadTimeout = 0 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid metrics server read timeout - above maximum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerReadTimeout = 301 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid metrics server write timeout - below minimum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerWriteTimeout = 0 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid metrics server write timeout - above maximum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerWriteTimeout = 301 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid metrics server idle timeout - below minimum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerIdleTimeout = 0 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid metrics server idle timeout - above maximum",
			cfg: func() Config {
				c := baseCfg
				c.MetricsServerIdleTimeout = 301 * time.Second
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
