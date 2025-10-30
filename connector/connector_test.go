package connector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	// 简化测试准备，避免导入 test 包造成循环依赖
	prepare(t)
	defer clean()

	err := Load(config.Conf)
	utils.Dump(config.Conf, "ERROR---", err, "-- END ERROR---")
	utils.Dump(
		"REDIS---",
		os.Getenv("REDIS_TEST_HOST"),
		os.Getenv("REDIS_TEST_PORT"),
		os.Getenv("REDIS_TEST_USER"),
		os.Getenv("REDIS_TEST_PASS"),
		"-- END REDIS---",
	)

	utils.Dump(
		"SQLITE---",
		os.Getenv("SQLITE_DB"),
		"-- END SQLITE---",
	)

	if err != nil {
		t.Fatal(err)
	}
	check(t)
}

// prepare 简化版的测试准备函数
func prepare(t *testing.T) {
	root := os.Getenv("YAO_DEV")
	if root == "" {
		root, _ = os.Getwd()
		root = filepath.Join(root, "..")
	}
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
}

// clean 简化版的清理函数
func clean() {
	connector.Connectors = map[string]connector.Connector{}
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range connector.Connectors {
		ids[id] = true
	}

	utils.Dump(ids)

	assert.True(t, ids["mongo"])
	assert.True(t, ids["mysql"])
	assert.True(t, ids["redis"])
	assert.True(t, ids["sqlite"])
}
