package config_test

import (
	"os"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnv_WithDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Set only required env vars
	os.Setenv("PORT", "8080")
	os.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("GRACEFUL_SHUTDOWN_TIMEOUT")
	}()

	// Unset optional vars to test defaults
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("HOST")

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.Equal(t, "INFO", envConfig.LogLevel)
	assert.Equal(t, "0.0.0.0", envConfig.Server.Host)
	assert.Equal(t, uint(8080), envConfig.Server.Port)
}

func TestLoadEnv_WithCustomValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Set custom env vars
	os.Setenv("PORT", "9090")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", "60s")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("HOST")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("GRACEFUL_SHUTDOWN_TIMEOUT")
	}()

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.Equal(t, "DEBUG", envConfig.LogLevel)
	assert.Equal(t, "127.0.0.1", envConfig.Server.Host)
	assert.Equal(t, uint(9090), envConfig.Server.Port)
}

func TestLoadEnv_ReadFromReplicas(t *testing.T) {
	t.Setenv("PRIVACY_CACHE_REDIS_READ_FROM_REPLICAS", "true")

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.True(t, envConfig.Privacy.RedisReadFromReplicas)
}

func TestLoadEnv_SingleflightAndJitter(t *testing.T) {
	t.Setenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED", "true")
	t.Setenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE", "0.15")

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.True(t, envConfig.Privacy.CacheSingleflightEnabled)
	assert.InDelta(t, 0.15, envConfig.Privacy.CacheTTLJitterPercentage, 1e-9)
}

func TestLoadEnv_SingleflightAndJitterDefaults(t *testing.T) {
	os.Unsetenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED")
	os.Unsetenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE")
	defer func() {
		os.Unsetenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED")
		os.Unsetenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE")
	}()

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.False(t, envConfig.Privacy.CacheSingleflightEnabled)
	assert.Zero(t, envConfig.Privacy.CacheTTLJitterPercentage)
}

func TestLoadEnv_InvalidPort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	os.Setenv("PORT", "invalid")
	os.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("GRACEFUL_SHUTDOWN_TIMEOUT")
	}()

	_, err := config.LoadEnv()

	require.Error(t, err)
}

func TestLoadEnv_LogLevelCaseSensitivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "debug", "info", "warn", "error"}

	for _, level := range logLevels {
		t.Run(level, func(t *testing.T) {
			os.Setenv("PORT", "8080")
			os.Setenv("LOG_LEVEL", level)
			os.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")
			defer func() {
				os.Unsetenv("PORT")
				os.Unsetenv("LOG_LEVEL")
				os.Unsetenv("GRACEFUL_SHUTDOWN_TIMEOUT")
			}()

			envConfig, err := config.LoadEnv()

			require.NoError(t, err)
			assert.Equal(t, level, envConfig.LogLevel)
		})
	}
}
