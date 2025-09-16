package image

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageManager(t *testing.T) {
	tempDir := t.TempDir()
	maxCacheSize := int64(100 * 1024 * 1024) // 100MB

	manager := NewImageManager(tempDir, maxCacheSize)
	defer manager.Close()

	assert.NotNil(t, manager.cacheManager)
	assert.NotNil(t, manager.renderer)
	assert.NotNil(t, manager.downloader)
	assert.Equal(t, maxCacheSize, manager.maxCacheSize)
	assert.Equal(t, time.Hour, manager.cleanupInterval)
	assert.Len(t, manager.supportedFormats, 6) // JPEG, PNG, GIF, WebP, BMP, SVG
}

func TestImageManager_IsImageFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/bmp", true},
		{"image/svg+xml", true},
		{"text/plain", false},
		{"application/octet-stream", false},
		{"video/mp4", false},
		{"", false},
		{"image/unknown", false},
	}

	for _, test := range tests {
		t.Run(test.contentType, func(t *testing.T) {
			result := manager.IsImageFile(test.contentType)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestImageManager_GetSupportedFormats(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	formats := manager.GetSupportedFormats()
	expectedFormats := []ImageFormat{FormatJPEG, FormatPNG, FormatGIF, FormatWebP, FormatBMP, FormatSVG}

	assert.Equal(t, len(expectedFormats), len(formats))
	for _, expected := range expectedFormats {
		assert.Contains(t, formats, expected)
	}
}

func TestImageManager_GetCurrentPreview(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	// 初始状态应该没有预览
	preview := manager.GetCurrentPreview()
	assert.Nil(t, preview)

	// 手动设置预览
	testPreview := &ImagePreview{
		FileKey:    "test.jpg",
		FilePath:   "/tmp/test.jpg",
		Format:     FormatJPEG,
		CreateTime: time.Now(),
	}
	manager.currentPreview = testPreview

	preview = manager.GetCurrentPreview()
	assert.NotNil(t, preview)
	assert.Equal(t, testPreview, preview)
}

func TestImageManager_ClearPreview(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	// 设置一个预览
	testPreview := &ImagePreview{
		FileKey:    "test.jpg",
		FilePath:   "/tmp/test.jpg",
		Format:     FormatJPEG,
		CreateTime: time.Now(),
	}
	manager.currentPreview = testPreview

	// 清除预览
	err := manager.ClearPreview()
	assert.NoError(t, err)
	assert.Nil(t, manager.currentPreview)
}

func TestImageManager_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	// 初始统计
	stats := manager.GetStats()
	assert.Equal(t, int64(0), stats.TotalPreviews)
	assert.Equal(t, int64(0), stats.CacheHits)
	assert.Equal(t, int64(0), stats.CacheMisses)
	assert.Equal(t, float64(0), stats.CacheHitRate)
	assert.NotEmpty(t, stats.RendererType)
	assert.IsType(t, bool(false), stats.SupportGraphics)

	// 模拟一些预览活动
	manager.previewCount = 10
	manager.cacheHitCount = 7
	manager.cacheMissCount = 3

	stats = manager.GetStats()
	assert.Equal(t, int64(10), stats.TotalPreviews)
	assert.Equal(t, int64(7), stats.CacheHits)
	assert.Equal(t, int64(3), stats.CacheMisses)
	assert.Equal(t, float64(70), stats.CacheHitRate)
}

func TestImageManager_CalculateDisplaySize(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	tests := []struct {
		name         string
		originalSize ImageSize
		// 注意：实际的最大尺寸取决于渲染器的能力
		// 这里我们测试逻辑而不是具体数值
	}{
		{
			name:         "small image",
			originalSize: ImageSize{Width: 100, Height: 100},
		},
		{
			name:         "large image",
			originalSize: ImageSize{Width: 2000, Height: 1500},
		},
		{
			name:         "wide image",
			originalSize: ImageSize{Width: 3000, Height: 500},
		},
		{
			name:         "tall image",
			originalSize: ImageSize{Width: 500, Height: 3000},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			displaySize := manager.calculateDisplaySize(test.originalSize)

			// 显示尺寸应该是有效的
			assert.Greater(t, displaySize.Width, 0)
			assert.Greater(t, displaySize.Height, 0)

			// 如果原图较小，显示尺寸可能相等
			// 如果原图较大，显示尺寸应该被缩放
			caps := manager.renderer.GetCapabilities()
			if test.originalSize.Width <= caps.MaxWidth && test.originalSize.Height <= caps.MaxHeight {
				assert.Equal(t, test.originalSize, displaySize)
			} else {
				// 至少一个维度应该被缩放
				assert.True(t, displaySize.Width <= caps.MaxWidth)
				assert.True(t, displaySize.Height <= caps.MaxHeight)
			}
		})
	}
}

func TestImageManager_GetImageInfo(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	tests := []struct {
		filename       string
		expectedFormat ImageFormat
		expectError    bool
	}{
		{"test.jpg", FormatJPEG, false},
		{"test.jpeg", FormatJPEG, false},
		{"test.png", FormatPNG, false},
		{"test.gif", FormatGIF, false},
		{"test.webp", FormatWebP, false},
		{"test.bmp", FormatBMP, false},
		{"test.svg", FormatSVG, false},
		{"test.txt", "", true},
		{"test.unknown", "", true},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			// 创建测试文件
			testFile := filepath.Join(tempDir, test.filename)
			err := os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)

			format, size, err := manager.getImageInfo(testFile)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedFormat, format)
				assert.Greater(t, size.Width, 0)
				assert.Greater(t, size.Height, 0)
			}
		})
	}
}

func TestImageManager_PreviewImageUnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	ctx := context.Background()
	fileInfo := FileItem{
		Key:         "test.txt",
		Size:        100,
		ContentType: "text/plain",
		Category:    "document",
	}

	preview, err := manager.PreviewImage(ctx, "test.txt", fileInfo)
	assert.Error(t, err)
	assert.Nil(t, preview)

	// 检查错误类型
	var formatErr *FormatError
	assert.ErrorAs(t, err, &formatErr)
	assert.Equal(t, "text/plain", formatErr.Format)
}

func TestImageManager_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	// 测试清理不会出错
	err := manager.Cleanup()
	assert.NoError(t, err)
}

func TestImageManager_Close(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)

	// 设置一个当前预览
	testPreview := &ImagePreview{
		FileKey:    "test.jpg",
		FilePath:   "/tmp/test.jpg",
		Format:     FormatJPEG,
		CreateTime: time.Now(),
	}
	manager.currentPreview = testPreview

	// 关闭管理器
	err := manager.Close()
	assert.NoError(t, err)

	// 验证预览已被清除
	assert.Nil(t, manager.currentPreview)
}

func TestImageManager_SetBucketName(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	bucketName := "test-bucket"
	manager.SetBucketName(bucketName)

	// 验证下载器的存储桶名称已设置
	assert.Equal(t, bucketName, manager.downloader.bucketName)
}

// 基准测试
func BenchmarkImageManager_IsImageFile(b *testing.B) {
	tempDir := b.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	contentTypes := []string{
		"image/jpeg",
		"image/png",
		"text/plain",
		"application/octet-stream",
		"image/gif",
		"video/mp4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contentType := contentTypes[i%len(contentTypes)]
		manager.IsImageFile(contentType)
	}
}

func BenchmarkImageManager_GetStats(b *testing.B) {
	tempDir := b.TempDir()
	manager := NewImageManager(tempDir, 10*1024*1024)
	defer manager.Close()

	// 模拟一些活动
	manager.previewCount = 1000
	manager.cacheHitCount = 700
	manager.cacheMissCount = 300

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetStats()
	}
}
