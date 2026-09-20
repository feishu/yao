package share

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

func TestDBConnectAndPoolSettings(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "test_yao_pool.db")
	defer os.Remove(dbFile)

	dbCfg := config.Database{
		Driver:          "sqlite3",
		Primary:         []string{dbFile},
		MaxIdleConns:    5,
		MaxOpenConns:    20,
		ConnMaxIdleTime: 60,
		ConnMaxLifetime: 600,
	}

	err := DBConnect(dbCfg)
	assert.NoError(t, err)
	assert.NotNil(t, capsule.Global)

	// Verify connections exist
	var count int
	capsule.Global.Connections.Range(func(key, value any) bool {
		count++
		return true
	})
	assert.GreaterOrEqual(t, count, 1)

	// Test close
	time.Sleep(100 * time.Millisecond)
	err = DBClose()
	assert.NoError(t, err)
}

func TestDBCloseNilSafe(t *testing.T) {
	orig := capsule.Global
	defer func() {
		capsule.Global = orig
	}()

	capsule.Global = nil
	err := DBClose()
	assert.NoError(t, err)
}

