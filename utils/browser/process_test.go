package browser

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	goufs "github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/gou/process"
)

func TestProcessPDFSupportsStreamOutput(t *testing.T) {
	Init()

	previous := pdfRenderer
	pdfRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("pdf-data"), nil
	}
	defer func() {
		pdfRenderer = previous
	}()

	res, err := process.New("utils.browser.pdf", "<html><body>hello</body></html>", map[string]interface{}{
		"output": "stream",
	}).Exec()

	assert.NoError(t, err)
	assert.Equal(t, []byte("pdf-data"), res)
}

func TestProcessPDFRequiresFilenameForFileOutput(t *testing.T) {
	Init()

	err := process.New("utils.browser.pdf", "<html><body>hello</body></html>", map[string]interface{}{
		"output": "file",
	}).Execute()

	assert.NotNil(t, err)
	assert.Equal(t, "Exception|400: filename is required when output is file", err.Error())
}

func TestProcessPDFReturnsBase64(t *testing.T) {
	Init()

	previous := pdfRenderer
	pdfRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("pdf-data"), nil
	}
	defer func() {
		pdfRenderer = previous
	}()

	res, err := process.New("utils.browser.pdf", "<html><body>hello</body></html>").Exec()
	assert.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("pdf-data")), res)
}

func TestProcessPDFAcceptsJSONStringOptions(t *testing.T) {
	Init()

	previous := pdfRenderer
	pdfRenderer = func(html string, options Options) ([]byte, error) {
		assert.Equal(t, outputBase64, options.Output)
		return []byte("pdf-data"), nil
	}
	defer func() {
		pdfRenderer = previous
	}()

	res, err := process.New("utils.browser.pdf", "<html><body>hello</body></html>", "{\"output\":\"base64\"}").Exec()
	assert.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("pdf-data")), res)
}

func TestProcessPNGWritesFileToDataFS(t *testing.T) {
	Init()

	previous := pngRenderer
	pngRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("png-data"), nil
	}
	defer func() {
		pngRenderer = previous
	}()

	tempDir := t.TempDir()
	previousData, hasData := goufs.FileSystems["data"]
	goufs.Register("data", system.New(tempDir))
	defer func() {
		if hasData {
			goufs.Register("data", previousData)
			return
		}
		delete(goufs.FileSystems, "data")
	}()

	res, err := process.New("utils.browser.png", "<html><body>hello</body></html>", map[string]interface{}{
		"output":   "file",
		"filename": "images/test.png",
	}).Exec()
	assert.NoError(t, err)

	result, ok := res.(Result)
	assert.True(t, ok)
	assert.Equal(t, "images/test.png", result.Filename)
	assert.Equal(t, "image/png", result.ContentType)

	content, readErr := os.ReadFile(filepath.Join(tempDir, "images", "test.png"))
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("png-data"), content)
}
