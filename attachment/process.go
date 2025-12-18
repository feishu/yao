package attachment

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func getContext(process *process.Process) context.Context {
	if process.Context != nil {
		return process.Context
	}
	return context.Background()
}

func init() {
	process.RegisterGroup("attachment", map[string]process.Handler{
		"upload":          processUpload,
		"download":        processDownload,
		"read":            processRead,
		"readBase64":      processReadBase64,
		"info":            processInfo,
		"list":            processList,
		"base64":          processBase64,
		"getPresignedUrl": processGetURL,
	})
}

// processUpload attachments.Upload uploader file [options]
// file can be a path in data fs or bytes
func processUpload(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileArg := process.Args[1]

	var reader io.Reader
	var filename string
	var size int64

	switch v := fileArg.(type) {
	case string:
		// Assume it's a path in data fs
		dataFS, err := fs.Get("data")
		if err != nil {
			exception.New("data filesystem not found", 500).Throw()
		}

		content, err := dataFS.ReadFile(v)
		if err != nil {
			exception.New("failed to read file %s: %v", 500, v, err).Throw()
		}

		reader = bytes.NewReader(content)
		filename = filepath.Base(v)
		size = int64(len(content))

	case []byte:
		reader = bytes.NewReader(v)
		filename = "upload.bin"
		size = int64(len(v))

	default:
		exception.New("invalid file argument", 400).Throw()
	}

	options := UploadOption{}
	if process.NumOfArgs() > 2 {
		data := process.ArgsMap(2)
		if v, ok := data["compress_image"].(bool); ok {
			options.CompressImage = v
		}
		if v, ok := data["compress_size"].(float64); ok {
			options.CompressSize = int(v)
		}
		if v, ok := data["gzip"].(bool); ok {
			options.Gzip = v
		}
		if v, ok := data["original_filename"].(string); ok {
			options.OriginalFilename = v
			filename = v
		}
		if v, ok := data["public"].(bool); ok {
			options.Public = v
		}
		if v, ok := data["share"].(string); ok {
			options.Share = v
		}
	}

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	// Create a mock FileHeader
	header := &FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: filename,
			Size:     size,
		},
	}

	res, err := manager.Upload(getContext(process), header, reader, options)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

// processDownload attachments.Download uploader fileID
func processDownload(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileID := process.ArgsString(1)

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	res, err := manager.Download(getContext(process), fileID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// processRead attachments.Read uploader fileID
func processRead(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileID := process.ArgsString(1)

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	res, err := manager.Read(getContext(process), fileID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// processReadBase64 attachments.ReadBase64 uploader fileID
func processReadBase64(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileID := process.ArgsString(1)

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	res, err := manager.ReadBase64(getContext(process), fileID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// processInfo attachments.Info uploader fileID
func processInfo(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileID := process.ArgsString(1)

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	res, err := manager.Info(getContext(process), fileID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// processList attachments.List uploader [options]
func processList(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	uploader := process.ArgsString(0)

	option := ListOption{}
	if process.NumOfArgs() > 1 {
		data := process.ArgsMap(1)
		if v, ok := data["page"].(float64); ok {
			option.Page = int(v)
		}
		if v, ok := data["page_size"].(float64); ok {
			option.PageSize = int(v)
		}
		if v, ok := data["order_by"].(string); ok {
			option.OrderBy = v
		}
		if v, ok := data["filters"].(map[string]interface{}); ok {
			option.Filters = v
		}
	}

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	res, err := manager.List(getContext(process), option)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// processBase64 attachments.Base64 value [dataURI]
func processBase64(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	value := process.ArgsString(0)
	dataURI := false
	if process.NumOfArgs() > 1 {
		dataURI = process.ArgsBool(1)
	}
	return Base64(getContext(process), value, dataURI)
}

// processGetURL attachments.GetURL uploader fileID [contentType]
func processGetURL(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	uploader := process.ArgsString(0)
	fileID := process.ArgsString(1)

	contentType := ""
	if process.NumOfArgs() > 2 {
		contentType = process.ArgsString(2)
	}

	manager, err := Select(uploader)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	return manager.GetPresignedUrl(getContext(process), fileID, contentType)
}
