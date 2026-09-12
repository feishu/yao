package attachment

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func setupTestLocalUploader(t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", "yao_attachment_test_*")
	assert.NoError(t, err)

	uploaderName := fmt.Sprintf("test_local_%d", os.Getpid())
	option := ManagerOption{
		Driver:  "local",
		MaxSize: "50M",
		Options: map[string]interface{}{
			"path": tempDir,
		},
	}
	_, err = Register(uploaderName, option.Driver, option)
	assert.NoError(t, err)

	return uploaderName, tempDir
}

func TestAttachmentRawObjectProcesses(t *testing.T) {
	uploader, tempDir := setupTestLocalUploader(t)
	defer os.RemoveAll(tempDir)

	t.Run("attachment.put and attachment.get with bytes", func(t *testing.T) {
		content := []byte("hello raw object storage")
		key := "docs/hello.txt"

		// 1. Put raw bytes directly
		p := process.New("attachment.put", uploader, key, content, "text/plain")
		uploadedKey := p.Run()
		assert.Equal(t, key, uploadedKey)

		// 2. Verify exists
		existsP := process.New("attachment.exists", uploader, key)
		assert.True(t, existsP.Run().(bool))

		// 3. Get raw bytes directly
		getP := process.New("attachment.get", uploader, key)
		retrieved := getP.Run()
		assert.Equal(t, content, retrieved)

		// 4. URL / getUrl
		urlP := process.New("attachment.url", uploader, key)
		urlVal := urlP.Run().(string)
		assert.NotEmpty(t, urlVal)

		getUrlP := process.New("attachment.getUrl", uploader, key)
		assert.Equal(t, urlVal, getUrlP.Run().(string))

		// 5. Delete raw object
		delP := process.New("attachment.delete", uploader, key)
		delP.Run()

		// 6. Verify not exists
		existsAfterDel := process.New("attachment.exists", uploader, key)
		assert.False(t, existsAfterDel.Run().(bool))
	})

	t.Run("attachment.put with base64 data URI", func(t *testing.T) {
		// "world" in base64 is "d29ybGQ="
		base64DataURI := "data:image/png;base64,d29ybGQ="
		key := "images/world.png"

		p := process.New("attachment.put", uploader, key, base64DataURI)
		uploadedKey := p.Run()
		assert.Equal(t, key, uploadedKey)

		getP := process.New("attachment.get", uploader, key)
		assert.Equal(t, []byte("world"), getP.Run())

		// Verify physical file was written in local tempDir
		localFile := filepath.Join(tempDir, key)
		assert.FileExists(t, localFile)
	})
}

func TestAttachmentManagerConcurrency(t *testing.T) {
	const goroutines = 30
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			uploaderName := fmt.Sprintf("concurrent_uploader_%d", id%5)

			for j := 0; j < iterations; j++ {
				// Concurrent Register
				Register(uploaderName, "local", ManagerOption{
					Driver: "local",
					Options: map[string]interface{}{
						"path": "/tmp",
					},
				})

				// Concurrent Select
				m, err := Select(uploaderName)
				if err == nil && m != nil {
					_ = m.Name
				}

				// Concurrent Count
				_ = Count()

				// Concurrent Range
				Range(func(name string, manager *Manager) bool {
					return true
				})
			}
		}(i)
	}

	wg.Wait()
}
