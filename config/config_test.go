package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestNewConfig(t *testing.T) {
// 	cfg := NewConfig()
// 	var vBool = func(name string) bool {
// 		if name == "true" || name == "1" {
// 			return true
// 		}
// 		return false
// 	}

// 	xiangPath := os.Getenv("XIANG_PATH")
// 	if xiangPath == "" {
// 		xiangPath = "bin://xiang"
// 	}

// 	assert.Equal(t, cfg.Mode, os.Getenv("XIANG_MODE"))
// 	assert.Equal(t, cfg.Root, os.Getenv("XIANG_ROOT"))
// 	assert.Equal(t, cfg.Path, xiangPath)

// 	assert.Equal(t, cfg.Service.Debug, vBool(os.Getenv("XIANG_SERVICE_DEBUG")))
// 	assert.Equal(t, strings.Join(cfg.Service.Allow, "|"), os.Getenv("XIANG_SERVICE_ALLOW"))
// 	assert.Equal(t, cfg.Service.Host, os.Getenv("XIANG_SERVICE_HOST"))
// 	assert.Equal(t, cfg.Service.Port, any.Of(os.Getenv("XIANG_SERVICE_PORT")).CInt())

// 	assert.Equal(t, cfg.Database.Debug, vBool(os.Getenv("XIANG_DB_DEBUG")))
// 	assert.Equal(t, strings.Join(cfg.Database.Primary, "|"), os.Getenv("XIANG_DB_PRIMARY"))
// 	assert.Equal(t, strings.Join(cfg.Database.Secondary, "|"), os.Getenv("XIANG_DB_SECONDARY"))
// 	assert.Equal(t, cfg.Database.AESKey, os.Getenv("XIANG_DB_AESKEY"))

// 	assert.Equal(t, cfg.JWT.Secret, os.Getenv("XIANG_JWT_SECRET"))

// 	assert.Equal(t, cfg.Log.Access, os.Getenv("XIANG_LOG_ACCESS"))
// 	assert.Equal(t, cfg.Log.Error, os.Getenv("XIANG_LOG_ERROR"))
// 	assert.Equal(t, cfg.Log.DB, os.Getenv("XIANG_LOG_DB"))
// 	assert.Equal(t, cfg.Log.Plugin, os.Getenv("XIANG_LOG_PLUGIN"))

// }

// func TestNewConfigFrom(t *testing.T) {
// 	assert.True(t, true)
// 	assert.True(t, true)
// }

func TestLoadFrom(t *testing.T) {
	cfg := LoadFrom(filepath.Join(os.Getenv("YAO_DEV"), ".env"))
	root, _ := filepath.Abs(os.Getenv("YAO_ROOT"))
	assert.Equal(t, cfg.Root, root)
	assert.Equal(t, cfg.Mode, os.Getenv("YAO_ENV"))
	assert.Equal(t, cfg.Host, os.Getenv("YAO_HOST"))
	assert.Equal(t, fmt.Sprintf("%d", cfg.Port), os.Getenv("YAO_PORT"))
	assert.Equal(t, cfg.JWTSecret, os.Getenv("YAO_JWT_SECRET"))
	assert.Equal(t, cfg.Log, os.Getenv("YAO_LOG"))
	assert.Equal(t, cfg.LogMode, os.Getenv("YAO_LOG_MODE"))
	assert.Equal(t, cfg.DB.Driver, os.Getenv("YAO_DB_DRIVER"))
	assert.Equal(t, cfg.DB.Primary[0], os.Getenv("YAO_DB_PRIMARY"))
	// assert.Equal(t, cfg.DB.Secondary[0], os.Getenv("YAO_DB_SECONDARY"))
}

func TestLoadDefaultsInspectSourceContentOn(t *testing.T) {
	withUnsetEnv(t, "YAO_RUNTIME_INSPECT_SOURCE_CONTENT")

	cfg := Load()
	if assert.NotNil(t, cfg.Runtime.InspectSourceContent) {
		assert.True(t, *cfg.Runtime.InspectSourceContent)
	}
}

func TestLoadCanDisableInspectSourceContent(t *testing.T) {
	t.Setenv("YAO_RUNTIME_INSPECT_SOURCE_CONTENT", "false")

	cfg := Load()
	if assert.NotNil(t, cfg.Runtime.InspectSourceContent) {
		assert.False(t, *cfg.Runtime.InspectSourceContent)
	}
}

func TestLoadDefaultsPProfOff(t *testing.T) {
	withUnsetEnv(t, "YAO_PPROF_ENABLED")
	withUnsetEnv(t, "YAO_PPROF_HOST")
	withUnsetEnv(t, "YAO_PPROF_PORT")

	cfg := Load()
	assert.False(t, cfg.PProf.Enabled)
	assert.Equal(t, "127.0.0.1", cfg.PProf.Host)
	assert.Equal(t, 6060, cfg.PProf.Port)
}

func TestLoadCanEnablePProf(t *testing.T) {
	t.Setenv("YAO_PPROF_ENABLED", "true")
	t.Setenv("YAO_PPROF_HOST", "127.0.0.2")
	t.Setenv("YAO_PPROF_PORT", "6061")

	cfg := Load()
	assert.True(t, cfg.PProf.Enabled)
	assert.Equal(t, "127.0.0.2", cfg.PProf.Host)
	assert.Equal(t, 6061, cfg.PProf.Port)
}

func withUnsetEnv(t *testing.T, key string) {
	t.Helper()

	old, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}
