package script

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	yruntime "github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range v8.Scripts {
		ids[id] = true
	}
	assert.True(t, ids["tests.task.mail"])
	assert.True(t, ids["tests.api"])
	assert.True(t, ids["runtime.basic"])
	assert.True(t, ids["runtime.bridge"])
	assert.True(t, ids["__yao_service.foo"])
}

func TestLoadAggregatesErrorsAndContinues(t *testing.T) {
	cfg, cleanup := prepareScriptLoadApp(t, map[string]string{
		"scripts/good.ts":         "export function Good() { return 'ok' }\n",
		"scripts/bad.ts":          "export const broken = ;\n",
		"services/good.ts":        "export function Service() { return 'ok' }\n",
		"services/bad-service.ts": "export const serviceBroken = ;\n",
	})
	defer cleanup()

	err := Load(cfg)
	var loadErrs *LoadErrors
	require.ErrorAs(t, err, &loadErrs)
	require.Len(t, loadErrs.Items, 2)

	files := []string{loadErrs.Items[0].File, loadErrs.Items[1].File}
	sort.Strings(files)
	assert.Equal(t, []string{
		"scripts/bad.ts",
		"services/bad-service.ts",
	}, files)

	assert.NotZero(t, loadErrs.Items[0].Line+loadErrs.Items[1].Line)
	assert.NotEmpty(t, loadErrs.Items[0].Text)
	assert.NotNil(t, LastLoadErrors())

	_, has := v8.Scripts["good"]
	assert.True(t, has)
	_, has = v8.Scripts["__yao_service.good"]
	assert.True(t, has)
	_, has = v8.Scripts["bad"]
	assert.False(t, has)
	_, has = v8.Scripts["__yao_service.bad_service"]
	assert.False(t, has)
}

func TestLoadDoesNotFailWhenServicesDirMissing(t *testing.T) {
	cfg, cleanup := prepareScriptLoadApp(t, map[string]string{
		"scripts/good.ts": "export function Good() { return 'ok' }\n",
	})
	defer cleanup()

	err := Load(cfg)
	require.NoError(t, err)

	_, has := v8.Scripts["good"]
	assert.True(t, has)
}

func prepareScriptLoadApp(t *testing.T, files map[string]string) (config.Config, func()) {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "data"), 0o755))

	for name, content := range files {
		filename := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
	}

	oldApp := application.App
	oldScripts := v8.Scripts
	oldRootScripts := v8.RootScripts

	app, err := application.OpenFromDisk(root)
	require.NoError(t, err)
	application.Load(app)

	cfg := config.Conf
	cfg.Root = root
	cfg.AppSource = root
	cfg.DataRoot = filepath.Join(root, "data")

	require.NoError(t, yruntime.Start(cfg))

	cleanup := func() {
		yruntime.Stop()
		application.App = oldApp
		v8.Scripts = oldScripts
		v8.RootScripts = oldRootScripts
	}

	return cfg, cleanup
}
