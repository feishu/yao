package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/runtime/await"
)

// JSLoader provides functionality to load and execute JavaScript files in v8go
type JSLoader struct {
	script *v8.Script
}

// NewJSLoader creates a new JSLoader instance with a simple script
func NewJSLoader() (*JSLoader, error) {
	// Create a simple empty script
	script, err := v8.MakeScript([]byte(""), "__jsloader__", 5*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create v8 script: %w", err)
	}

	return &JSLoader{
		script: script,
	}, nil
}

// NewJSLoaderWithScript creates a new JSLoader with an existing v8 script
func NewJSLoaderWithScript(script *v8.Script) *JSLoader {
	return &JSLoader{
		script: script,
	}
}

// LoadFile loads a JavaScript file from the application filesystem
// file: relative path from app root (e.g., "scripts/hello.js")
// id: unique identifier for the loaded script (e.g., "scripts.hello")
func (l *JSLoader) LoadFile(file string, id string) (*v8.Script, error) {
	script, err := v8.Load(file, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load file %s: %w", file, err)
	}

	return script, nil
}

// LoadFileWithPath loads a JavaScript file from an absolute file path
// path: absolute file path (e.g., "/path/to/script.js")
// id: unique identifier for the loaded script
func (l *JSLoader) LoadFileWithPath(path string, id string) (*v8.Script, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	script, err := v8.MakeScript(data, id, 5*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to compile script %s: %w", id, err)
	}

	return script, nil
}

// ExecuteScript executes JavaScript code string
// The code should define a function that will be called
// code: JavaScript code to execute (should define and export functions)
// filename: name used for error reporting (e.g., "inline.js")
// Returns the compiled script, use script.NewContext() to get a context for calling functions
func (l *JSLoader) ExecuteScript(code string, filename string) (*v8.Script, error) {
	script, err := v8.MakeScript([]byte(code), filename, 5*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to compile script %s: %w", filename, err)
	}

	return script, nil
}

// LoadFiles loads multiple JavaScript files from the application filesystem
// files: map of file paths to their IDs
// Returns a map of IDs to loaded scripts
func (l *JSLoader) LoadFiles(files map[string]string) (map[string]*v8.Script, error) {
	scripts := make(map[string]*v8.Script)
	for file, id := range files {
		script, err := l.LoadFile(file, id)
		if err != nil {
			return nil, err
		}
		scripts[id] = script
	}
	return scripts, nil
}

// LoadDirectory loads all JavaScript files from a directory
// dir: directory path relative to app root
// pattern: file pattern to match (e.g., "*.js", "*.ts")
// Returns a map of IDs to loaded scripts
func (l *JSLoader) LoadDirectory(dir string, pattern string) (map[string]*v8.Script, error) {
	scripts := make(map[string]*v8.Script)
	exts := []string{pattern}
	
	err := application.App.Walk(dir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// Generate ID from file path
		id := generateID(root, file)
		script, err := l.LoadFile(file, id)
		if err != nil {
			return err
		}
		scripts[id] = script
		return nil
	}, exts...)

	if err != nil {
		return nil, err
	}

	return scripts, nil
}

// GetScript returns the underlying v8 script
func (l *JSLoader) GetScript() *v8.Script {
	return l.script
}

// Close releases resources (no-op for scripts)
func (l *JSLoader) Close() error {
	// Scripts are managed by the v8 runtime pool
	return nil
}

// LoadGlobalScript loads a JavaScript file from the application filesystem
// and registers it in the global v8 runtime pool
func LoadGlobalScript(file string, id string) (*v8.Script, error) {
	script, err := v8.Load(file, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load global script %s: %w", file, err)
	}
	return script, nil
}

// ExecuteCode executes JavaScript code and returns the result by calling a function
// The code must define a function with the given name
// code: JavaScript code that defines functions
// filename: name used for error reporting (e.g., "inline.js")
// functionName: name of the function to call after loading
// args: arguments to pass to the function
func ExecuteCode(code string, filename string, functionName string, args ...interface{}) (interface{}, error) {
	script, err := v8.MakeScript([]byte(code), filename, 5*time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to compile script %s: %w", filename, err)
	}

	ctx, err := script.NewContext("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create context for %s: %w", filename, err)
	}
	defer ctx.Close()

	// Register syncAwait function
	await.Register(ctx)

	result, err := ctx.Call(functionName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to call function %s in %s: %w", functionName, filename, err)
	}

	return result, nil
}

// generateID generates a unique ID from root and file path
func generateID(root, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		rel = file
	}

	// Remove extension
	id := rel[:len(rel)-len(filepath.Ext(rel))]

	// Replace path separators with dots
	id = filepath.ToSlash(id)
	id = filepath.Join(filepath.Split(id))
	
	return id
}
