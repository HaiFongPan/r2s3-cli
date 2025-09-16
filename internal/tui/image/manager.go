package image

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

// ImageManager 管理图片预览的整个生命周期
type ImageManager struct {
	cacheManager   *CacheManager
	renderer       ImageRendererInterface
	downloader     *ImageDownloader
	currentPreview *ImagePreview

	// 配置和状态
	maxCacheSize     int64
	cleanupInterval  time.Duration
	supportedFormats []ImageFormat

	// 统计信息
	previewCount   int64
	cacheHitCount  int64
	cacheMissCount int64

	// display constraints
	maxDisplayWidth  int
	maxDisplayHeight int
	// text mode (ANSI) constraints
	cellCols int
	cellRows int
	useText  bool
}

// ImageManagerInterface 定义图片管理器接口
type ImageManagerInterface interface {
	PreviewImage(ctx context.Context, fileKey string, fileInfo FileItem) (*ImagePreview, error)
	PreviewImageForce(ctx context.Context, fileKey string, fileInfo FileItem) (*ImagePreview, error)
	PreviewImageAt(ctx context.Context, fileKey string, fileInfo FileItem, startCol, startRow int) (*ImagePreview, error)
	PreviewImageAtForce(ctx context.Context, fileKey string, fileInfo FileItem, startCol, startRow int) (*ImagePreview, error)
	ClearPreview() error
	IsImageFile(contentType string) bool
	GetSupportedFormats() []ImageFormat
	GetCurrentPreview() *ImagePreview
	GetStats() ImageManagerStats
	Cleanup() error
	Close() error
}

// FileItem 表示文件项信息（从 TUI 包导入的类型）
type FileItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	Category     string
}

// NewImageManager 创建新的图片管理器实例
func NewImageManager(cacheDir string, maxCacheSize int64) *ImageManager {
	return &ImageManager{
		cacheManager:     NewCacheManager(cacheDir, maxCacheSize),
		renderer:         NewImageRenderer(),
		downloader:       NewImageDownloader(),
		maxCacheSize:     maxCacheSize,
		cleanupInterval:  time.Hour,
		supportedFormats: []ImageFormat{FormatJPEG, FormatPNG, FormatGIF, FormatWebP, FormatBMP, FormatSVG},
		maxDisplayWidth:  0,
		maxDisplayHeight: 0,
		cellCols:         0,
		cellRows:         0,
		useText:          true,
	}
}

// SetDisplaySize sets the maximum display size in pixels for image rendering
func (m *ImageManager) SetDisplaySize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	m.maxDisplayWidth = width
	m.maxDisplayHeight = height
	m.renderer.SetMaxSize(width, height)
}

// SetCellSize sets text-mode cell dimensions
func (m *ImageManager) SetCellSize(cols, rows int) {
	if cols > 0 {
		m.cellCols = cols
	}
	if rows > 0 {
		m.cellRows = rows
	}
	// inform renderer when using text mode
	m.renderer.SetCellSize(m.cellCols, m.cellRows)
}

// SetUseTextRender enables/disables ANSI text rendering mode
func (m *ImageManager) SetUseTextRender(enabled bool) {
	m.useText = enabled
	m.renderer.SetTextMode(enabled)
	if enabled {
		// ensure renderer has cell size
		m.renderer.SetCellSize(m.cellCols, m.cellRows)
	}
}

// PreviewImage 生成图片预览
func (m *ImageManager) PreviewImage(ctx context.Context, fileKey string, fileInfo FileItem) (*ImagePreview, error) {
	return m.generatePreview(ctx, fileKey, fileInfo, false)
}

// PreviewImageForce 强制跳过缓存生成图片预览
func (m *ImageManager) PreviewImageForce(ctx context.Context, fileKey string, fileInfo FileItem) (*ImagePreview, error) {
	return m.generatePreview(ctx, fileKey, fileInfo, true)
}

// PreviewImageAt 在指定单元格位置生成图片预览（Kitty 协议）
func (m *ImageManager) PreviewImageAt(ctx context.Context, fileKey string, fileInfo FileItem, startCol, startRow int) (*ImagePreview, error) {
	return m.previewImageAt(ctx, fileKey, fileInfo, startCol, startRow, false)
}

// PreviewImageAtForce 在指定位置强制重新渲染图片预览
func (m *ImageManager) PreviewImageAtForce(ctx context.Context, fileKey string, fileInfo FileItem, startCol, startRow int) (*ImagePreview, error) {
	return m.previewImageAt(ctx, fileKey, fileInfo, startCol, startRow, true)
}

func (m *ImageManager) generatePreview(ctx context.Context, fileKey string, fileInfo FileItem, force bool) (*ImagePreview, error) {
	startTime := time.Now()

	// 增加预览计数
	m.previewCount++

	// 检查格式
	if !m.IsImageFile(fileInfo.ContentType) {
		return nil, &FormatError{
			Format:   fileInfo.ContentType,
			FilePath: fileKey,
			Reason:   "unsupported image format",
		}
	}

	var (
		localPath string
		err       error
		cacheHit  bool
	)

	if !force {
		if cachedPath, hit, getErr := m.cacheManager.Get(fileKey); hit && getErr == nil {
			cacheHit = true
			m.cacheHitCount++
			localPath = cachedPath
		} else {
			cacheHit = false
		}
	}

	if localPath == "" {
		m.cacheMissCount++
		if force {
			if err := m.cacheManager.Delete(fileKey); err != nil {
				logrus.WithError(err).WithField("file_key", fileKey).Warn("failed to invalidate cache before forced preview")
			}
		}

		downloadedPath, err := m.downloader.DownloadWithProgress(ctx, fileKey, nil)
		if err != nil {
			return nil, &NetworkError{Op: "download", Err: err, Code: 0}
		}

		cachedPath, err := m.cacheManager.Put(fileKey, downloadedPath)
		if err != nil {
			localPath = downloadedPath
		} else {
			localPath = cachedPath
		}
		cacheHit = false
	}

	format, originalSize, err := m.getImageInfo(localPath)
	if err != nil {
		return nil, err
	}

	var renderedData string
	if m.useText {
		if m.cellCols > 0 && m.cellRows > 0 {
			m.renderer.SetCellSize(m.cellCols, m.cellRows)
		}
		renderedData, err = m.renderer.RenderImage(localPath, 0, 0)
	} else {
		maxW, maxH := 0, 0
		if m.maxDisplayWidth > 0 && m.maxDisplayHeight > 0 {
			maxW, maxH = m.maxDisplayWidth, m.maxDisplayHeight
		}
		renderedData, err = m.renderer.RenderImage(localPath, maxW, maxH)
	}
	if err != nil {
		renderedData = fmt.Sprintf("🖼️ Image file: %s\n⚠️  Could not render image: %v", filepath.Base(localPath), err)
	}

	displaySize := m.calculateDisplaySize(originalSize)
	renderCols, renderRows := m.estimateCellUsage(displaySize)

	preview := &ImagePreview{
		FileKey:      fileKey,
		FilePath:     localPath,
		OriginalSize: originalSize,
		DisplaySize:  displaySize,
		Format:       format,
		RenderedData: renderedData,
		RenderCols:   renderCols,
		RenderRows:   renderRows,
		CacheHit:     cacheHit,
		LoadTime:     time.Since(startTime),
		CreateTime:   time.Now(),
	}

	m.currentPreview = preview

	logrus.WithFields(logrus.Fields{
		"file_key":  fileKey,
		"cache_hit": cacheHit,
		"force":     force,
		"load_ms":   preview.LoadTime.Milliseconds(),
	}).Info("image preview generated")

	return preview, nil
}

func (m *ImageManager) previewImageAt(ctx context.Context, fileKey string, fileInfo FileItem, startCol, startRow int, force bool) (*ImagePreview, error) {
	p, err := m.generatePreview(ctx, fileKey, fileInfo, force)
	if err != nil {
		return nil, err
	}
	cols, rows := m.estimateCellUsage(p.DisplaySize)
	placed, err := m.renderer.RenderImageAtCells(p.FilePath, cols, rows, startCol, startRow)
	if err != nil {
		return nil, err
	}
	p.RenderedData = placed
	p.RenderCols = cols
	p.RenderRows = rows
	m.currentPreview = p
	return p, nil
}

// ClearPreview 清除当前预览
func (m *ImageManager) ClearPreview() error {
	if m.currentPreview != nil {
		// 文本模式无需清屏；图形协议下删除图像
		if !m.useText {
			_ = m.renderer.ClearScreen()
		}
		m.currentPreview = nil
	}
	return nil
}

// Close 关闭管理器并清理资源
func (m *ImageManager) Close() error {
	// 清除当前预览
	if err := m.ClearPreview(); err != nil {
		return err
	}

	// 停止缓存管理器的自动清理
	if m.cacheManager != nil {
		m.cacheManager.StopAutoCleanup()
	}

	return nil
}

// getImageInfo 获取图片信息（格式和尺寸）
func (m *ImageManager) getImageInfo(imagePath string) (ImageFormat, ImageSize, error) {
	// 从文件扩展名推断格式
	ext := strings.ToLower(filepath.Ext(imagePath))
	var format ImageFormat

	switch ext {
	case ".jpg", ".jpeg":
		format = FormatJPEG
	case ".png":
		format = FormatPNG
	case ".gif":
		format = FormatGIF
	case ".webp":
		format = FormatWebP
	case ".bmp":
		format = FormatBMP
	case ".svg":
		format = FormatSVG
	default:
		return "", ImageSize{}, &FormatError{
			Format:   ext,
			FilePath: imagePath,
			Reason:   "unknown file extension",
		}
	}

	// 解码图片头部，尽量拿到真实尺寸（对 webp/svg 可能失败，则回退默认）
	size := ImageSize{Width: 800, Height: 600}
	if f, err := os.Open(imagePath); err == nil {
		defer f.Close()
		if cfg, _, err2 := image.DecodeConfig(f); err2 == nil {
			if cfg.Width > 0 && cfg.Height > 0 {
				size = ImageSize{Width: cfg.Width, Height: cfg.Height}
			}
		}
	}

	return format, size, nil
}

// calculateDisplaySize 计算显示尺寸
func (m *ImageManager) calculateDisplaySize(originalSize ImageSize) ImageSize {
	caps := m.renderer.GetCapabilities()
	maxWidth := caps.MaxWidth
	maxHeight := caps.MaxHeight

	if originalSize.Width <= maxWidth && originalSize.Height <= maxHeight {
		return originalSize
	}

	// 计算缩放比例，保持宽高比
	scaleX := float64(maxWidth) / float64(originalSize.Width)
	scaleY := float64(maxHeight) / float64(originalSize.Height)

	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	return ImageSize{
		Width:  int(float64(originalSize.Width) * scale),
		Height: int(float64(originalSize.Height) * scale),
	}
}

// estimateCellUsage 根据显示像素尺寸估算终端单元占用（与渲染器算法保持一致）
func (m *ImageManager) estimateCellUsage(displaySize ImageSize) (int, int) {
	// 近似每个字符 8x16 px；字符高宽比约 1:2
	const charW = 8.0
	const charH = 16.0
	const charRatio = 2.0

	maxW := m.maxDisplayWidth
	maxH := m.maxDisplayHeight
	if maxW <= 0 || maxH <= 0 {
		// fallback to renderer caps
		caps := m.renderer.GetCapabilities()
		maxW = caps.MaxWidth
		maxH = caps.MaxHeight
	}

	availableCols := float64(maxW) / charW
	availableRows := float64(maxH) / charH

	imgCols := float64(displaySize.Width) / charW
	imgRows := float64(displaySize.Height) / charH * charRatio

	scaleX := availableCols / imgCols
	scaleY := availableRows / imgRows
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	if scale > 1.0 {
		scale = 1.0
	}

	cols := int(imgCols * scale)
	rows := int((imgRows * scale) / charRatio)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

// SetDownloaderClient 设置下载器的 S3 客户端
func (m *ImageManager) SetDownloaderClient(client interface{}) error {
	if s3Client, ok := client.(*s3.Client); ok {
		m.downloader.SetS3Client(s3Client)
		return nil
	}
	return fmt.Errorf("invalid client type")
}

// SetBucketName 设置存储桶名称
func (m *ImageManager) SetBucketName(bucketName string) {
	m.downloader.SetBucketName(bucketName)
}

// IsImageFile 检查文件是否为支持的图片格式
func (m *ImageManager) IsImageFile(contentType string) bool {
	if contentType == "" {
		return false
	}

	// 标准化内容类型
	mainType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	// 检查主类型
	if !strings.HasPrefix(mainType, "image/") {
		return false
	}

	// 检查具体的图片格式
	supportedTypes := map[string]bool{
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/bmp":     true,
		"image/svg+xml": true,
	}

	return supportedTypes[mainType]
}

// GetSupportedFormats 返回支持的图片格式列表
func (m *ImageManager) GetSupportedFormats() []ImageFormat {
	return m.supportedFormats
}

// GetCurrentPreview 获取当前预览
func (m *ImageManager) GetCurrentPreview() *ImagePreview {
	return m.currentPreview
}

// GetStats 获取管理器统计信息
func (m *ImageManager) GetStats() ImageManagerStats {
	cacheStats := m.cacheManager.GetCacheStats()

	hitRate := float64(0)
	if m.previewCount > 0 {
		hitRate = float64(m.cacheHitCount) / float64(m.previewCount) * 100
	}

	stats := ImageManagerStats{
		TotalPreviews:   m.previewCount,
		CacheHits:       m.cacheHitCount,
		CacheMisses:     m.cacheMissCount,
		CacheHitRate:    hitRate,
		CacheStats:      cacheStats,
		RendererType:    "unknown",
		SupportGraphics: m.renderer.IsSupported(),
	}

	// 尝试获取渲染器类型
	if imgRenderer, ok := m.renderer.(*ImageRenderer); ok {
		stats.RendererType = string(imgRenderer.TerminalType)
	}

	return stats
}

// Cleanup 清理资源
func (m *ImageManager) Cleanup() error {
	if m.cacheManager != nil {
		return m.cacheManager.Cleanup()
	}
	return nil
}
