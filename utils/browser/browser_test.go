package browser

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	goufs "github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/system"
)

func TestRenderPDFReturnsBase64(t *testing.T) {
	previous := pdfRenderer
	pdfRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("pdf-data"), nil
	}
	defer func() {
		pdfRenderer = previous
	}()

	res, err := RenderPDF("<html><body>hello</body></html>", Options{Output: outputBase64})
	assert.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("pdf-data")), res)
}

func TestRenderPDFWritesFileToDataFS(t *testing.T) {
	previous := pdfRenderer
	pdfRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("pdf-data"), nil
	}
	defer func() {
		pdfRenderer = previous
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

	res, err := RenderPDF("<html><body>hello</body></html>", Options{
		Output:   outputFile,
		Filename: "reports/test.pdf",
	})
	assert.NoError(t, err)

	result, ok := res.(Result)
	assert.True(t, ok)
	assert.Equal(t, "reports/test.pdf", result.Filename)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Equal(t, len([]byte("pdf-data")), result.Size)

	content, readErr := os.ReadFile(filepath.Join(tempDir, "reports", "test.pdf"))
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("pdf-data"), content)
}

func TestRenderPNGWritesFileToAbsolutePath(t *testing.T) {
	previous := pngRenderer
	pngRenderer = func(html string, options Options) ([]byte, error) {
		return []byte("png-data"), nil
	}
	defer func() {
		pngRenderer = previous
	}()

	filename := filepath.Join(t.TempDir(), "images", "test.png")
	res, err := RenderPNG("<html><body>hello</body></html>", Options{
		Output:   outputFile,
		Filename: filename,
	})
	assert.NoError(t, err)

	result, ok := res.(Result)
	assert.True(t, ok)
	assert.Equal(t, filename, result.Filename)
	assert.Equal(t, "image/png", result.ContentType)
	assert.Equal(t, len([]byte("png-data")), result.Size)

	content, readErr := os.ReadFile(filename)
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("png-data"), content)
}

func TestDiscoverBrowserBinaryUsesEnvPath(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "chrome")
	writeErr := os.WriteFile(tempFile, []byte("browser"), 0755)
	assert.NoError(t, writeErr)

	t.Setenv("YAO_BROWSER_BIN", tempFile)

	path, err := discoverBrowserBinary()
	assert.NoError(t, err)
	assert.Equal(t, tempFile, path)
}

func TestDiscoverBrowserBinaryRejectsMissingEnvPath(t *testing.T) {
	t.Setenv("YAO_BROWSER_BIN", filepath.Join(t.TempDir(), "missing-browser"))

	_, err := discoverBrowserBinary()
	assert.EqualError(t, err, "browser binary configured by YAO_BROWSER_BIN is not accessible: "+os.Getenv("YAO_BROWSER_BIN"))
}

func TestApplyBaseURLInjectsIntoHTMLTagWithAttributes(t *testing.T) {
	html := `<html lang="zh-CN"><body><img src="logo.png"></body></html>`
	result := applyBaseURL(html, "https://example.com/assets/")

	assert.Contains(t, result, `<head><base href="https://example.com/assets/"></head>`)
	assert.Contains(t, result, `<html lang="zh-CN">`)
}

func TestRenderPDFWithRodRenderer(t *testing.T) {
	path, err := discoverBrowserBinary()
	if err != nil {
		t.Skipf("browser unavailable: %v", err)
	}
	if path == "" {
		t.Skip("browser binary not found on current machine")
	}

	content, err := renderPDFWithRod(`
		<html>
			<body style="font-family: sans-serif;">
				<h1>Hello Yao</h1>
				<p>PDF smoke test</p>
			</body>
		</html>
	`, Options{
		Output:          outputBase64,
		Timeout:         10000,
		PrintBackground: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestRenderPNGWithRodRendererUsesWhiteBackgroundByDefault(t *testing.T) {
	path, err := discoverBrowserBinary()
	if err != nil {
		t.Skipf("browser unavailable: %v", err)
	}
	if path == "" {
		t.Skip("browser binary not found on current machine")
	}

	content, err := renderPNGWithRod(`
		<html>
			<body>
				<div>Hello Yao</div>
			</body>
		</html>
	`, Options{
		Output:  outputBase64,
		Timeout: 10000,
		Width:   200,
		Height:  100,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	img, decodeErr := png.Decode(bytes.NewReader(content))
	assert.NoError(t, decodeErr)

	red, green, blue, alpha := img.At(0, 0).RGBA()
	assert.Equal(t, uint32(0xffff), red)
	assert.Equal(t, uint32(0xffff), green)
	assert.Equal(t, uint32(0xffff), blue)
	assert.Equal(t, uint32(0xffff), alpha)
}
