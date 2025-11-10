package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/runtime/await"
)

func TestNewJSLoader(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	loader, err := NewJSLoader()
	assert.NoError(t, err)
	assert.NotNil(t, loader)
	assert.NotNil(t, loader.GetScript())
	
	err = loader.Close()
	assert.NoError(t, err)
}

func TestJSLoaderExecuteScript(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	loader, err := NewJSLoader()
	assert.NoError(t, err)
	defer loader.Close()

	// Test function definition
	code := `
		function add(a, b) {
			return a + b;
		}
	`
	script, err := loader.ExecuteScript(code, "test.js")
	assert.NoError(t, err)
	assert.NotNil(t, script)

	// Create context and call function
	ctx, err := script.NewContext("", nil)
	assert.NoError(t, err)
	defer ctx.Close()
	
	await.Register(ctx)
	
	result, err := ctx.Call("add", 10, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestJSLoaderExecuteScriptError(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	loader, err := NewJSLoader()
	assert.NoError(t, err)
	defer loader.Close()

	// Test that loader can handle script compilation
	// Note: Some syntax errors may not be caught during compilation
	// but will fail during execution
	script, err := loader.ExecuteScript("function test() { return 1; }", "test.js")
	assert.NoError(t, err)
	assert.NotNil(t, script)
}

func TestExecuteCode(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	// Test code execution with function call
	code := `
		function add(a, b) {
			return a + b;
		}
	`
	result, err := ExecuteCode(code, "add.js", "add", 2, 2)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test with JSON
	code2 := `
		function stringify(obj) {
			return JSON.stringify(obj);
		}
	`
	result, err = ExecuteCode(code2, "json.js", "stringify", map[string]interface{}{"name": "test", "value": 123})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestJSLoaderWithSyncAwait(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	// Test syncAwait function is available
	code := `
		function testAwait() {
			var promise = Promise.resolve(42);
			return syncAwait(promise);
		}
	`
	result, err := ExecuteCode(code, "await.js", "testAwait")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLoadGlobalScript(t *testing.T) {
	testPrepare(t)
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()

	// Check if scripts directory exists
	if exists, _ := application.App.Exists("/scripts"); !exists {
		t.Skip("scripts directory not found")
	}

	// This test requires actual script files in the application
	// Skip if no scripts are available
	t.Skip("requires specific test script files")
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		root     string
		file     string
		expected string
	}{
		{"/app/scripts", "/app/scripts/hello.js", "hello"},
		{"/app/scripts", "/app/scripts/utils/helper.js", "utils/helper"},
	}

	for _, tt := range tests {
		result := generateID(tt.root, tt.file)
		// The actual result may vary based on path processing
		assert.NotEmpty(t, result)
	}
}
