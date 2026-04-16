package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
)

func TestRegisteredScriptNamesSorted(t *testing.T) {
	oldScripts := v8.Scripts
	t.Cleanup(func() {
		v8.Scripts = oldScripts
	})

	v8.Scripts = map[string]*v8.Script{
		"beta":             {ID: "beta"},
		"alpha":            {ID: "alpha"},
		"__yao_service.db": {ID: "__yao_service.db"},
	}

	assert.Equal(t, []string{
		"__yao_service.db",
		"alpha",
		"beta",
	}, registeredScriptNames())
}

func TestScriptsCmdOutputsAllRegisteredScripts(t *testing.T) {
	oldScripts := v8.Scripts
	oldLoader := loadRegisteredScripts
	oldOutput := scriptsCmd.OutOrStdout()
	oldErr := scriptsCmd.ErrOrStderr()
	t.Cleanup(func() {
		v8.Scripts = oldScripts
		loadRegisteredScripts = oldLoader
		scriptsCmd.SetOut(oldOutput)
		scriptsCmd.SetErr(oldErr)
	})

	loadRegisteredScripts = func() error {
		return nil
	}

	v8.Scripts = map[string]*v8.Script{
		"zeta":                 {ID: "zeta"},
		"app.user.login":       {ID: "app.user.login"},
		"__yao_service.health": {ID: "__yao_service.health"},
	}

	var stdout bytes.Buffer
	scriptsCmd.SetOut(&stdout)
	scriptsCmd.SetErr(io.Discard)

	err := scriptsCmd.RunE(scriptsCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "__yao_service.health\napp.user.login\nzeta\n", stdout.String())
}
