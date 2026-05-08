package browser

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	goufs "github.com/yaoapp/gou/fs"
)

var pdfRenderer = func(html string, options Options) ([]byte, error) {
	return nil, fmt.Errorf("browser pdf renderer is not implemented")
}

var pngRenderer = func(html string, options Options) ([]byte, error) {
	return nil, fmt.Errorf("browser png renderer is not implemented")
}

// RenderPDF 渲染 HTML 为 PDF
func RenderPDF(html string, options Options) (interface{}, error) {
	options, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}

	if html == "" {
		return nil, newValidationError("html content is required")
	}

	content, err := pdfRenderer(html, options)
	if err != nil {
		return nil, err
	}

	return buildOutput(content, options, "application/pdf")
}

// RenderPNG 渲染 HTML 为 PNG
func RenderPNG(html string, options Options) (interface{}, error) {
	options, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}

	if html == "" {
		return nil, newValidationError("html content is required")
	}

	content, err := pngRenderer(html, options)
	if err != nil {
		return nil, err
	}

	return buildOutput(content, options, "image/png")
}

func buildOutput(content []byte, options Options, contentType string) (interface{}, error) {
	if options.Output == outputBase64 {
		return base64.StdEncoding.EncodeToString(content), nil
	}

	if options.Output == outputStream {
		return content, nil
	}

	if filepath.IsAbs(options.Filename) {
		if err := os.MkdirAll(filepath.Dir(options.Filename), os.ModePerm); err != nil && !os.IsExist(err) {
			return nil, err
		}

		if err := os.WriteFile(options.Filename, content, 0644); err != nil {
			return nil, err
		}

		return Result{
			Filename:    options.Filename,
			ContentType: contentType,
			Size:        len(content),
		}, nil
	}

	dataFS, err := goufs.Get("data")
	if err != nil {
		return nil, err
	}

	if _, err := goufs.WriteFile(dataFS, options.Filename, content, 0644); err != nil {
		return nil, err
	}

	return Result{
		Filename:    options.Filename,
		ContentType: contentType,
		Size:        len(content),
	}, nil
}
