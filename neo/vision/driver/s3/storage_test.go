package s3

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestS3Storage(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("Create Storage", func(t *testing.T) {
		options := map[string]interface{}{
			"endpoint":    os.Getenv("S3_API"),
			"region":      "auto",
			"key":         os.Getenv("S3_ACCESS_KEY"),
			"secret":      os.Getenv("S3_SECRET_KEY"),
			"bucket":      os.Getenv("S3_BUCKET"),
			"prefix":      "vision-test",
			"expiration":  10 * time.Minute,
			"compression": true,
		}

		storage, err := New(options)
		if err != nil {
			t.Logf("Error creating storage: %v", err)
		}
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		if storage != nil {
			assert.Equal(t, os.Getenv("S3_API"), storage.Endpoint)
			assert.Equal(t, "auto", storage.Region)
			assert.Equal(t, os.Getenv("S3_ACCESS_KEY"), storage.Key)
			assert.Equal(t, os.Getenv("S3_SECRET_KEY"), storage.Secret)
			assert.Equal(t, os.Getenv("S3_BUCKET"), storage.Bucket)
			assert.Equal(t, "vision-test", storage.prefix)
			assert.Equal(t, 10*time.Minute, storage.Expiration)
			assert.True(t, storage.compression)
		}
	})

	t.Run("Upload and Download Image with Compression", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":    os.Getenv("S3_API"),
			"region":      "auto",
			"key":         os.Getenv("S3_ACCESS_KEY"),
			"secret":      os.Getenv("S3_SECRET_KEY"),
			"bucket":      os.Getenv("S3_BUCKET"),
			"prefix":      "vision-test",
			"expiration":  5 * time.Minute,
			"compression": true,
		})
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		// Create a test image (2000x2000 pixels)
		img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		fileID, err := storage.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Download and verify size
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify dimensions
		bounds := downloadedImg.Bounds()
		assert.LessOrEqual(t, bounds.Dx(), MaxImageSize)
		assert.LessOrEqual(t, bounds.Dy(), MaxImageSize)
	})

	t.Run("Upload Image without Compression", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":    os.Getenv("S3_API"),
			"region":      "auto",
			"key":         os.Getenv("S3_ACCESS_KEY"),
			"secret":      os.Getenv("S3_SECRET_KEY"),
			"bucket":      os.Getenv("S3_BUCKET"),
			"prefix":      "vision-test",
			"expiration":  5 * time.Minute,
			"compression": false,
		})
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		// Create a test image (2000x2000 pixels)
		img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		fileID, err := storage.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Download and verify size
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify dimensions are unchanged
		bounds := downloadedImg.Bounds()
		assert.Equal(t, 2000, bounds.Dx())
		assert.Equal(t, 2000, bounds.Dy())
	})

	t.Run("Upload and Download Text File", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":    os.Getenv("S3_API"),
			"region":      "auto",
			"key":         os.Getenv("S3_ACCESS_KEY"),
			"secret":      os.Getenv("S3_SECRET_KEY"),
			"bucket":      os.Getenv("S3_BUCKET"),
			"prefix":      "vision-test",
			"expiration":  5 * time.Minute,
			"compression": true,
		})
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		content := []byte("test content")
		reader := bytes.NewReader(content)
		fileID, err := storage.Upload(context.Background(), "test.txt", reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Get presigned URL
		url := storage.URL(context.Background(), fileID)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, "X-Amz-Signature")
		assert.Contains(t, url, "X-Amz-Expires")

		// Download
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		if err != nil {
			t.Logf("Download error: %v", err)
			t.FailNow()
		}
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		if reader2 != nil {
			downloaded, err := io.ReadAll(reader2)
			assert.NoError(t, err)
			assert.Equal(t, content, downloaded)
			reader2.Close()
		}
	})

	t.Run("Download Non-existent File", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":    os.Getenv("S3_API"),
			"region":      "auto",
			"key":         os.Getenv("S3_ACCESS_KEY"),
			"secret":      os.Getenv("S3_SECRET_KEY"),
			"bucket":      os.Getenv("S3_BUCKET"),
			"prefix":      "vision-test",
			"expiration":  5 * time.Minute,
			"compression": true,
		})
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		_, _, err = storage.Download(context.Background(), "non-existent.txt")
		assert.Error(t, err)
	})
}

func TestS3OBSConfiguration(t *testing.T) {
	t.Run("OBS Domain Auto-Detection", func(t *testing.T) {
		options := map[string]interface{}{
			"endpoint": "https://obs.cn-north-4.myhuaweicloud.com",
			"key":      "test-ak",
			"secret":   "test-sk",
			"bucket":   "my-obs-bucket",
		}
		storage, err := New(options)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.NotNil(t, storage.UsePathStyle)
		assert.False(t, *storage.UsePathStyle, "OBS should default to Virtual-Hosted style (UsePathStyle=false)")
		assert.Equal(t, "cn-north-4", storage.Region, "Region should be inferred from OBS endpoint")
	})

	t.Run("Explicit Provider OBS", func(t *testing.T) {
		options := map[string]interface{}{
			"provider": "obs",
			"endpoint": "custom-obs.example.com",
			"region":   "cn-east-3",
			"key":      "test-ak",
			"secret":   "test-sk",
			"bucket":   "my-obs-bucket",
		}
		storage, err := New(options)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.NotNil(t, storage.UsePathStyle)
		assert.False(t, *storage.UsePathStyle, "Provider OBS should force UsePathStyle=false")
		assert.Equal(t, "cn-east-3", storage.Region)
	})

	t.Run("Default MinIO Backward Compatibility", func(t *testing.T) {
		options := map[string]interface{}{
			"endpoint": "http://127.0.0.1:9000",
			"key":      "minioadmin",
			"secret":   "minioadmin",
			"bucket":   "minio-bucket",
		}
		storage, err := New(options)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.NotNil(t, storage.UsePathStyle)
		assert.True(t, *storage.UsePathStyle, "Default/MinIO should maintain UsePathStyle=true")
		assert.Equal(t, "auto", storage.Region)
	})

	t.Run("Endpoint Sanitization", func(t *testing.T) {
		options := map[string]interface{}{
			"provider": "obs",
			"endpoint": "https://my-obs-bucket.obs.cn-north-4.myhuaweicloud.com/",
			"key":      "test-ak",
			"secret":   "test-sk",
			"bucket":   "my-obs-bucket",
		}
		storage, err := New(options)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.False(t, *storage.UsePathStyle)
		assert.Equal(t, "cn-north-4", storage.Region)

		url := storage.URL(context.Background(), "image.png")
		assert.Contains(t, url, "my-obs-bucket.obs.cn-north-4.myhuaweicloud.com/image.png")
	})
}

