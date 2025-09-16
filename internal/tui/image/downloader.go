package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ProgressCallback 下载进度回调函数类型
type ProgressCallback func(downloaded, total int64)

// ImageDownloader 处理图片文件的下载
type ImageDownloader struct {
	s3Client         *s3.Client
	bucketName       string
	httpClient       *http.Client
	progressCallback ProgressCallback

	// 状态管理
	activeDownloads map[string]*DownloadState
	mutex           sync.RWMutex
	maxRetries      int
	timeout         time.Duration
}

// ImageDownloaderInterface 定义下载器接口
type ImageDownloaderInterface interface {
	DownloadImage(ctx context.Context, fileKey string) (string, error)
	DownloadWithProgress(ctx context.Context, fileKey string, callback ProgressCallback) (string, error)
	Cancel(ctx context.Context) error
	SetS3Client(client *s3.Client)
	SetBucketName(bucketName string)
	GetDownloadState(fileKey string) (*DownloadState, bool)
	CancelDownload(fileKey string) error
}

// NewImageDownloader 创建新的图片下载器
func NewImageDownloader() *ImageDownloader {
	return &ImageDownloader{
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		activeDownloads: make(map[string]*DownloadState),
		maxRetries:      3,
		timeout:         30 * time.Second,
	}
}

// SetS3Client 设置 S3 客户端
func (d *ImageDownloader) SetS3Client(client *s3.Client) {
	d.s3Client = client
}

// SetBucketName 设置存储桶名称
func (d *ImageDownloader) SetBucketName(bucketName string) {
	d.bucketName = bucketName
}

// DownloadImage 下载图片文件到临时目录
func (d *ImageDownloader) DownloadImage(ctx context.Context, fileKey string) (string, error) {
	return d.DownloadWithProgress(ctx, fileKey, nil)
}

// DownloadWithProgress 带进度回调的图片下载
func (d *ImageDownloader) DownloadWithProgress(ctx context.Context, fileKey string, callback ProgressCallback) (string, error) {
	if d.s3Client == nil {
		return "", fmt.Errorf("S3 client not configured")
	}

	// 创建下载状态
	state := &DownloadState{
		FileKey:   fileKey,
		Status:    DownloadStatusStarted,
		StartTime: time.Now(),
		Progress:  &DownloadProgress{FileKey: fileKey, StartTime: time.Now()},
	}

	// 注册下载状态
	d.mutex.Lock()
	d.activeDownloads[fileKey] = state
	d.mutex.Unlock()

	// 确保在函数结束时清理状态
	defer func() {
		d.mutex.Lock()
		delete(d.activeDownloads, fileKey)
		d.mutex.Unlock()
	}()

	// 创建带超时的上下文
	downloadCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// 创建临时文件
	tempDir := os.TempDir()
	fileName := filepath.Base(fileKey)
	tempPath := filepath.Join(tempDir, "r2s3-cli-"+fileName)

	// 更新状态为下载中
	state.Status = DownloadStatusDownloading
	state.TempPath = tempPath

	// 从 S3 获取对象
	result, err := d.s3Client.GetObject(downloadCtx, &s3.GetObjectInput{
		Bucket: &d.bucketName,
		Key:    &fileKey,
	})
	if err != nil {
		state.Status = DownloadStatusFailed
		state.Error = err
		return "", err
	}
	defer result.Body.Close()

	// 更新文件大小信息
	if result.ContentLength != nil {
		state.Progress.Total = *result.ContentLength
	}

	// 创建本地文件
	file, err := os.Create(tempPath)
	if err != nil {
		state.Status = DownloadStatusFailed
		state.Error = err
		return "", err
	}
	defer file.Close()

	// 使用进度跟踪复制
	finalPath, err := d.copyWithProgressAndState(result.Body, file, result.ContentLength, callback, tempPath, state)
	if err != nil {
		state.Status = DownloadStatusFailed
		state.Error = err
		os.Remove(tempPath)
		return "", err
	}

	// 更新状态为完成
	state.Status = DownloadStatusCompleted
	state.CompletedTime = time.Now()
	state.Progress.Percentage = 100.0

	return finalPath, nil
}

// copyWithProgress 带进度跟踪的文件复制
func (d *ImageDownloader) copyWithProgress(src io.Reader, dst io.Writer, total *int64, callback ProgressCallback, tempPath string) (string, error) {
	var written int64
	totalSize := int64(0)
	if total != nil {
		totalSize = *total
	}

	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			os.Remove(tempPath)
			return "", err
		}

		if n == 0 {
			break
		}

		if _, err := dst.Write(buffer[:n]); err != nil {
			os.Remove(tempPath)
			return "", err
		}

		written += int64(n)
		if callback != nil {
			callback(written, totalSize)
		}
	}

	return tempPath, nil
}

// copyWithProgressAndState 带状态管理的进度跟踪复制
func (d *ImageDownloader) copyWithProgressAndState(src io.Reader, dst io.Writer, total *int64, callback ProgressCallback, tempPath string, state *DownloadState) (string, error) {
	var written int64
	totalSize := int64(0)
	if total != nil {
		totalSize = *total
	}

	buffer := make([]byte, 32*1024) // 32KB buffer
	startTime := time.Now()

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}

		if n == 0 {
			break
		}

		if _, err := dst.Write(buffer[:n]); err != nil {
			return "", err
		}

		written += int64(n)

		// 更新状态和进度
		elapsed := time.Since(startTime)
		var speed int64
		var eta time.Duration

		if elapsed.Seconds() > 0 {
			speed = int64(float64(written) / elapsed.Seconds())
			if speed > 0 && totalSize > written {
				remainingBytes := totalSize - written
				eta = time.Duration(float64(remainingBytes)/float64(speed)) * time.Second
			}
		}

		percentage := float64(0)
		if totalSize > 0 {
			percentage = float64(written) / float64(totalSize) * 100
		}

		// 更新进度信息
		state.Progress.Downloaded = written
		state.Progress.Total = totalSize
		state.Progress.Percentage = percentage
		state.Progress.Speed = speed
		state.Progress.ETA = eta
		state.Progress.CurrentTime = time.Now()

		// 调用回调函数
		if callback != nil {
			callback(written, totalSize)
		}
	}

	return tempPath, nil
}

// Cancel 取消下载操作
func (d *ImageDownloader) Cancel(ctx context.Context) error {
	// 取消操作通过 context 处理
	return nil
}

// GetDownloadState 获取下载状态
func (d *ImageDownloader) GetDownloadState(fileKey string) (*DownloadState, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	state, exists := d.activeDownloads[fileKey]
	return state, exists
}

// CancelDownload 取消特定文件的下载
func (d *ImageDownloader) CancelDownload(fileKey string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	state, exists := d.activeDownloads[fileKey]
	if !exists {
		return fmt.Errorf("download not found for key: %s", fileKey)
	}

	state.Status = DownloadStatusCancelled

	// 清理临时文件
	if state.TempPath != "" {
		os.Remove(state.TempPath)
	}

	delete(d.activeDownloads, fileKey)
	return nil
}

// GetAllDownloadStates 获取所有下载状态
func (d *ImageDownloader) GetAllDownloadStates() map[string]*DownloadState {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	states := make(map[string]*DownloadState)
	for key, state := range d.activeDownloads {
		states[key] = state
	}
	return states
}

// DownloadWithRetry 带重试机制的下载
func (d *ImageDownloader) DownloadWithRetry(ctx context.Context, fileKey string, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 计算退避时间 (指数退避)
		if attempt > 0 {
			backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second // 最大退避30秒
			}

			select {
			case <-time.After(backoffDuration):
				// 继续重试
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		// 尝试下载
		result, err := d.DownloadImage(ctx, fileKey)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 检查是否为可重试的错误
		if !d.isRetryableError(err) {
			break
		}

		// 检查是否还能重试
		if attempt >= maxRetries {
			break
		}
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetryableError 判断错误是否可重试
func (d *ImageDownloader) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())

	// 网络相关的可重试错误
	retryableErrors := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"temporary failure",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, retryableError := range retryableErrors {
		if strings.Contains(errorStr, retryableError) {
			return true
		}
	}

	return false
}

// GetDownloadProgress 获取下载进度信息
type DownloadProgress struct {
	FileKey     string
	Downloaded  int64
	Total       int64
	Percentage  float64
	Speed       int64 // bytes per second
	ETA         time.Duration
	StartTime   time.Time
	CurrentTime time.Time
}

// EstimateFileSize 估算文件大小（通过 HEAD 请求）
func (d *ImageDownloader) EstimateFileSize(ctx context.Context, fileKey string) (int64, error) {
	if d.s3Client == nil {
		return 0, fmt.Errorf("S3 client not configured")
	}

	result, err := d.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &d.bucketName,
		Key:    &fileKey,
	})
	if err != nil {
		return 0, err
	}

	if result.ContentLength != nil {
		return *result.ContentLength, nil
	}

	return 0, nil
}

// ProgressTrackingReader 带进度跟踪的 Reader
type ProgressTrackingReader struct {
	reader     io.Reader
	downloaded int64
	total      int64
	callback   ProgressCallback
	startTime  time.Time
}

// NewProgressTrackingReader 创建进度跟踪 Reader
func NewProgressTrackingReader(reader io.Reader, total int64, callback ProgressCallback) *ProgressTrackingReader {
	return &ProgressTrackingReader{
		reader:    reader,
		total:     total,
		callback:  callback,
		startTime: time.Now(),
	}
}

// Read 实现 io.Reader 接口
func (ptr *ProgressTrackingReader) Read(p []byte) (n int, err error) {
	n, err = ptr.reader.Read(p)
	if n > 0 {
		ptr.downloaded += int64(n)
		if ptr.callback != nil {
			ptr.callback(ptr.downloaded, ptr.total)
		}
	}
	return n, err
}

// GetProgress 获取当前进度信息
func (ptr *ProgressTrackingReader) GetProgress() DownloadProgress {
	currentTime := time.Now()
	elapsed := currentTime.Sub(ptr.startTime)

	var speed int64
	var eta time.Duration

	if elapsed.Seconds() > 0 {
		speed = int64(float64(ptr.downloaded) / elapsed.Seconds())
		if speed > 0 && ptr.total > ptr.downloaded {
			remainingBytes := ptr.total - ptr.downloaded
			eta = time.Duration(float64(remainingBytes)/float64(speed)) * time.Second
		}
	}

	percentage := float64(0)
	if ptr.total > 0 {
		percentage = float64(ptr.downloaded) / float64(ptr.total) * 100
	}

	return DownloadProgress{
		Downloaded:  ptr.downloaded,
		Total:       ptr.total,
		Percentage:  percentage,
		Speed:       speed,
		ETA:         eta,
		StartTime:   ptr.startTime,
		CurrentTime: currentTime,
	}
}

// DownloadState 下载状态
type DownloadState struct {
	FileKey       string
	Status        DownloadStatus
	Progress      *DownloadProgress
	Error         error
	StartTime     time.Time
	CompletedTime time.Time
	TempPath      string
}

// DownloadStatus 下载状态枚举
type DownloadStatus string

const (
	DownloadStatusPending     DownloadStatus = "pending"
	DownloadStatusStarted     DownloadStatus = "started"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
	DownloadStatusCancelled   DownloadStatus = "cancelled"
)

// IsActive 检查下载是否处于活跃状态
func (s *DownloadState) IsActive() bool {
	return s.Status == DownloadStatusPending ||
		s.Status == DownloadStatusStarted ||
		s.Status == DownloadStatusDownloading
}

// IsCompleted 检查下载是否已完成
func (s *DownloadState) IsCompleted() bool {
	return s.Status == DownloadStatusCompleted
}

// IsFailed 检查下载是否失败
func (s *DownloadState) IsFailed() bool {
	return s.Status == DownloadStatusFailed || s.Status == DownloadStatusCancelled
}

// GetDuration 获取下载持续时间
func (s *DownloadState) GetDuration() time.Duration {
	if s.Status == DownloadStatusCompleted {
		return s.CompletedTime.Sub(s.StartTime)
	}
	return time.Since(s.StartTime)
}
