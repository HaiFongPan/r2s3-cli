package image

import (
	"fmt"
	"time"
)

// ImageFormat 支持的图片格式
type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatGIF  ImageFormat = "gif"
	FormatWebP ImageFormat = "webp"
	FormatBMP  ImageFormat = "bmp"
	FormatSVG  ImageFormat = "svg"
)

// ImageSize 图片尺寸信息
type ImageSize struct {
	Width  int
	Height int
}

// ImagePreview 表示一个图片预览实例
type ImagePreview struct {
	FileKey      string
	FilePath     string // 本地缓存路径
	OriginalSize ImageSize
	DisplaySize  ImageSize
	Format       ImageFormat
	RenderedData string // 渲染后的终端输出数据
	RenderCols   int    // 实际渲染占用的列数（终端单元）
	RenderRows   int    // 实际渲染占用的行数（终端单元）
	CacheHit     bool
	LoadTime     time.Duration
	CreateTime   time.Time
}

// RenderError 渲染错误类型
type RenderError struct {
	Terminal string
	Protocol string
	Err      error
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("render error on %s terminal with %s protocol: %v", e.Terminal, e.Protocol, e.Err)
}

// CacheError 缓存错误类型
type CacheError struct {
	Operation string
	Path      string
	Err       error
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache error during %s operation on %s: %v", e.Operation, e.Path, e.Err)
}

// NetworkError 网络错误类型
type NetworkError struct {
	Op   string
	Err  error
	Code int
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s (code: %d): %v", e.Op, e.Code, e.Err)
}

// FormatError 文件格式错误类型
type FormatError struct {
	Format   string
	FilePath string
	Reason   string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("format error for %s file %s: %s", e.Format, e.FilePath, e.Reason)
}

// CacheStats 缓存统计信息
type CacheStats struct {
	TotalFiles   int
	TotalSize    int64
	MaxSize      int64
	UsagePercent float64
	OldestEntry  *CacheEntry
	NewestEntry  *CacheEntry
}

// CacheMetrics 详细的缓存指标
type CacheMetrics struct {
	TotalFiles        int
	TotalSize         int64
	MaxSize           int64
	UsagePercent      float64
	HitRate           float64
	AverageFileSize   float64
	AverageAccessTime time.Time
	OldestEntry       *CacheEntry
	NewestEntry       *CacheEntry
	LastCleanupTime   time.Time
}

// CacheEvent 缓存事件类型
type CacheEvent struct {
	Type      CacheEventType
	Key       string
	Timestamp time.Time
	Details   map[string]interface{}
}

// CacheEventType 缓存事件类型枚举
type CacheEventType string

const (
	CacheEventHit     CacheEventType = "hit"
	CacheEventMiss    CacheEventType = "miss"
	CacheEventPut     CacheEventType = "put"
	CacheEventDelete  CacheEventType = "delete"
	CacheEventCleanup CacheEventType = "cleanup"
	CacheEventExpire  CacheEventType = "expire"
)

// ImageManagerStats 图片管理器统计信息
type ImageManagerStats struct {
	TotalPreviews   int64
	CacheHits       int64
	CacheMisses     int64
	CacheHitRate    float64
	CacheStats      CacheStats
	RendererType    string
	SupportGraphics bool
}
