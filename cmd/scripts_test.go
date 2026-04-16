package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	scriptpkg "github.com/yaoapp/yao/script"
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

func TestScriptsCmdOutputsRegisteredScriptCount(t *testing.T) {
	oldScripts := v8.Scripts
	oldLoader := loadRegisteredScripts
	oldPatterns := append([]string(nil), scriptMatchPatterns...)
	oldOutput := scriptsCmd.OutOrStdout()
	oldErr := scriptsCmd.ErrOrStderr()
	t.Cleanup(func() {
		v8.Scripts = oldScripts
		loadRegisteredScripts = oldLoader
		scriptMatchPatterns = oldPatterns
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
	assert.Equal(t, "3\n", stdout.String())
}

func TestFilterScriptNamesExactMatch(t *testing.T) {
	names, err := filterScriptNames([]string{
		"a.b",
		"a.c",
		"b.a",
	}, []string{"a.b"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.b"}, names)
}

func TestFilterScriptNamesWildcardMatch(t *testing.T) {
	names, err := filterScriptNames([]string{
		"user.login",
		"user.logout",
		"admin.login",
		"health",
	}, []string{"user.*", "*.login"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"admin.login",
		"user.login",
		"user.logout",
	}, names)
}

func TestFilterScriptNamesMultiplePatterns(t *testing.T) {
	names, err := filterScriptNames([]string{
		"alpha.beta",
		"beta.gamma",
		"gamma.delta",
	}, []string{"alpha.beta", "*.delta"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"alpha.beta",
		"gamma.delta",
	}, names)
}

func TestFilterScriptNamesInvalidPattern(t *testing.T) {
	_, err := filterScriptNames([]string{
		"alpha.beta",
	}, []string{"["})
	require.Error(t, err)
}

func TestScriptsCmdOutputsMatchedRegisteredScripts(t *testing.T) {
	oldScripts := v8.Scripts
	oldLoader := loadRegisteredScripts
	oldPatterns := append([]string(nil), scriptMatchPatterns...)
	oldOutput := scriptsCmd.OutOrStdout()
	oldErr := scriptsCmd.ErrOrStderr()
	t.Cleanup(func() {
		v8.Scripts = oldScripts
		loadRegisteredScripts = oldLoader
		scriptMatchPatterns = oldPatterns
		scriptsCmd.SetOut(oldOutput)
		scriptsCmd.SetErr(oldErr)
	})

	loadRegisteredScripts = func() error {
		return nil
	}

	v8.Scripts = map[string]*v8.Script{
		"user.login":  {ID: "user.login"},
		"user.logout": {ID: "user.logout"},
		"admin.login": {ID: "admin.login"},
	}

	require.NoError(t, scriptsCmd.Flags().Set("match", "user.*"))

	var stdout bytes.Buffer
	scriptsCmd.SetOut(&stdout)
	scriptsCmd.SetErr(io.Discard)

	err := scriptsCmd.RunE(scriptsCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "user.login\nuser.logout\n", stdout.String())
}

func TestScriptsCmdOutputsOnlyScriptErrors(t *testing.T) {
	oldLoader := loadRegisteredScriptErrors
	oldGetErrors := getRegisteredScriptErrors
	oldErrorOnly := scriptErrorOnly
	oldOutput := scriptsCmd.OutOrStdout()
	oldErr := scriptsCmd.ErrOrStderr()
	t.Cleanup(func() {
		loadRegisteredScriptErrors = oldLoader
		getRegisteredScriptErrors = oldGetErrors
		scriptErrorOnly = oldErrorOnly
		scriptsCmd.SetOut(oldOutput)
		scriptsCmd.SetErr(oldErr)
		_ = scriptsCmd.Flags().Set("error", "false")
	})

	loadRegisteredScriptErrors = func() error {
		return &scriptpkg.LoadErrors{
			Items: []scriptpkg.LoadError{
				{File: "scripts/bad.ts", Line: 1, Column: 23, Text: "Unexpected \";\""},
				{File: "services/bad.ts", Line: 2, Column: 5, Text: "Expected \")\" but found end of file"},
			},
		}
	}

	getRegisteredScriptErrors = func() *scriptpkg.LoadErrors {
		return &scriptpkg.LoadErrors{
			Items: []scriptpkg.LoadError{
				{File: "scripts/bad.ts", Line: 1, Column: 23, Text: "Unexpected \";\""},
				{File: "services/bad.ts", Line: 2, Column: 5, Text: "Expected \")\" but found end of file"},
			},
		}
	}

	require.NoError(t, scriptsCmd.Flags().Set("error", "true"))

	var stdout bytes.Buffer
	scriptsCmd.SetOut(&stdout)
	scriptsCmd.SetErr(io.Discard)

	err := scriptsCmd.RunE(scriptsCmd, nil)
	require.Error(t, err)
	assert.Equal(t, "scripts/bad.ts:1:23 Unexpected \";\"\nservices/bad.ts:2:5 Expected \")\" but found end of file\n", stdout.String())
}
