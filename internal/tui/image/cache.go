package image

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// CacheManager 管理 /tmp 目录下的图片缓存
type CacheManager struct {
	cacheDir      string
	maxSize       int64
	cleanupPolicy CleanupPolicy
	index         map[string]*CacheEntry
	mutex         sync.RWMutex

	// 自动清理相关
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
}

// CacheManagerInterface 定义缓存管理器接口
type CacheManagerInterface interface {
	Get(key string) (string, bool, error)
	Put(key, sourcePath string) (string, error)
	Delete(key string) error
	GetSize() int64
	Cleanup() error
	SetMaxSize(size int64)
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key         string
	FilePath    string
	Size        int64
	AccessTime  time.Time
	CreateTime  time.Time
	OriginalKey string // S3 中的原始文件key
	ContentType string
	Checksum    string // 用于验证文件完整性
}

// CacheIndex 缓存索引
type CacheIndex struct {
	Entries     map[string]*CacheEntry
	TotalSize   int64
	LastCleanup time.Time
}

// CleanupPolicy 清理策略
type CleanupPolicy int

const (
	CleanupPolicyLRU CleanupPolicy = iota
	CleanupPolicySize
	CleanupPolicyAge
)

// NewCacheManager 创建新的缓存管理器
func NewCacheManager(cacheDir string, maxSize int64) *CacheManager {
	// 确保缓存目录存在
	os.MkdirAll(cacheDir, 0755)

	cm := &CacheManager{
		cacheDir:      cacheDir,
		maxSize:       maxSize,
		cleanupPolicy: CleanupPolicyLRU,
		index:         make(map[string]*CacheEntry),
	}

	// 加载现有的缓存索引
	cm.loadIndex()

	// 启动自动清理
	cm.startAutoCleanup()

	return cm
}

// Get 从缓存中获取文件
func (c *CacheManager) Get(key string) (string, bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.index[key]
	if !exists {
		return "", false, nil
	}

	// 检查文件是否仍存在
	if _, err := os.Stat(entry.FilePath); os.IsNotExist(err) {
		// 文件已被删除，从索引中移除
		delete(c.index, key)
		return "", false, nil
	}

	// 更新访问时间
	entry.AccessTime = time.Now()
	return entry.FilePath, true, nil
}

// Put 将文件放入缓存
func (c *CacheManager) Put(key, sourcePath string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 生成缓存文件名
	hash := md5.Sum([]byte(key))
	ext := filepath.Ext(sourcePath)
	filename := fmt.Sprintf("%x%s", hash, ext)
	cachePath := filepath.Join(c.cacheDir, filename)

	// 复制文件到缓存目录
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()

	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return "", err
	}
	defer cacheFile.Close()

	// 复制文件内容并计算校验和
	checksum, err := c.copyWithChecksum(sourceFile, cacheFile)
	if err != nil {
		os.Remove(cachePath)
		return "", err
	}

	// 获取文件信息
	stat, err := os.Stat(cachePath)
	if err != nil {
		return "", err
	}

	// 检测内容类型
	contentType, err := c.detectContentType(sourcePath)
	if err != nil {
		contentType = "application/octet-stream"
	}

	// 创建缓存条目
	entry := &CacheEntry{
		Key:         key,
		FilePath:    cachePath,
		Size:        stat.Size(),
		AccessTime:  time.Now(),
		CreateTime:  time.Now(),
		OriginalKey: key,
		ContentType: contentType,
		Checksum:    checksum,
	}

	c.index[key] = entry

	// 保存缓存索引
	if err := c.saveIndex(); err != nil {
		// 记录错误但不影响主流程
		// TODO: 添加日志记录
	}

	return cachePath, nil
}

// Delete 从缓存中删除文件
func (c *CacheManager) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.index[key]
	if !exists {
		return nil
	}

	// 删除文件
	os.Remove(entry.FilePath)
	delete(c.index, key)
	return nil
}

// GetSize 获取缓存总大小
func (c *CacheManager) GetSize() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var totalSize int64
	for _, entry := range c.index {
		totalSize += entry.Size
	}
	return totalSize
}

// Cleanup 清理缓存
func (c *CacheManager) Cleanup() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查是否需要清理
	currentSize := c.calculateTotalSize()
	if currentSize <= c.maxSize {
		return nil
	}

	// 根据清理策略执行清理
	switch c.cleanupPolicy {
	case CleanupPolicyLRU:
		return c.cleanupByLRU(currentSize)
	case CleanupPolicySize:
		return c.cleanupBySize(currentSize)
	case CleanupPolicyAge:
		return c.cleanupByAge()
	default:
		return c.cleanupByLRU(currentSize)
	}
}

// cleanupByLRU 根据 LRU 策略清理缓存
func (c *CacheManager) cleanupByLRU(currentSize int64) error {
	// 创建按访问时间排序的条目列表
	entries := make([]*CacheEntry, 0, len(c.index))
	for _, entry := range c.index {
		entries = append(entries, entry)
	}

	// 按访问时间排序 (最久未访问的在前)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AccessTime.Before(entries[j].AccessTime)
	})

	// 删除最久未访问的条目，直到大小满足要求
	sizeToRemove := currentSize - c.maxSize
	var removedSize int64

	for _, entry := range entries {
		if removedSize >= sizeToRemove {
			break
		}

		// 删除文件
		if err := os.Remove(entry.FilePath); err == nil {
			removedSize += entry.Size
			delete(c.index, entry.Key)
		}
	}

	return c.saveIndex()
}

// cleanupBySize 根据文件大小清理缓存
func (c *CacheManager) cleanupBySize(currentSize int64) error {
	// 创建按文件大小排序的条目列表 (大文件优先删除)
	entries := make([]*CacheEntry, 0, len(c.index))
	for _, entry := range c.index {
		entries = append(entries, entry)
	}

	// 按文件大小排序 (大文件在前)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	// 删除大文件，直到大小满足要求
	sizeToRemove := currentSize - c.maxSize
	var removedSize int64

	for _, entry := range entries {
		if removedSize >= sizeToRemove {
			break
		}

		// 删除文件
		if err := os.Remove(entry.FilePath); err == nil {
			removedSize += entry.Size
			delete(c.index, entry.Key)
		}
	}

	return c.saveIndex()
}

// cleanupByAge 根据文件年龄清理缓存
func (c *CacheManager) cleanupByAge() error {
	cutoffTime := time.Now().Add(-24 * time.Hour) // 删除超过24小时的文件

	for key, entry := range c.index {
		if entry.CreateTime.Before(cutoffTime) {
			if err := os.Remove(entry.FilePath); err == nil {
				delete(c.index, key)
			}
		}
	}

	return c.saveIndex()
}

// calculateTotalSize 计算当前缓存总大小
func (c *CacheManager) calculateTotalSize() int64 {
	var totalSize int64
	for _, entry := range c.index {
		totalSize += entry.Size
	}
	return totalSize
}

// copyWithChecksum 复制文件并计算校验和
func (c *CacheManager) copyWithChecksum(src io.Reader, dst io.Writer) (string, error) {
	hasher := sha256.New()
	multiWriter := io.MultiWriter(dst, hasher)

	_, err := io.Copy(multiWriter, src)
	if err != nil {
		return "", err
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return checksum, nil
}

// detectContentType 检测文件内容类型
func (c *CacheManager) detectContentType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 读取文件头用于检测
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}

// saveIndex 保存缓存索引到磁盘
func (c *CacheManager) saveIndex() error {
	indexPath := filepath.Join(c.cacheDir, ".cache_index.json")

	cacheIndex := &CacheIndex{
		Entries:     c.index,
		TotalSize:   c.calculateTotalSize(),
		LastCleanup: time.Now(),
	}

	data, err := json.MarshalIndent(cacheIndex, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

// loadIndex 从磁盘加载缓存索引
func (c *CacheManager) loadIndex() error {
	indexPath := filepath.Join(c.cacheDir, ".cache_index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 索引文件不存在，这是正常的（首次使用）
			return nil
		}
		return err
	}

	var cacheIndex CacheIndex
	if err := json.Unmarshal(data, &cacheIndex); err != nil {
		return err
	}

	c.index = cacheIndex.Entries
	if c.index == nil {
		c.index = make(map[string]*CacheEntry)
	}

	// 验证缓存文件是否仍然存在
	c.validateCache()
	return nil
}

// validateCache 验证缓存条目对应的文件是否存在
func (c *CacheManager) validateCache() {
	for key, entry := range c.index {
		if _, err := os.Stat(entry.FilePath); os.IsNotExist(err) {
			delete(c.index, key)
		}
	}
}

// GetCacheStats 获取缓存统计信息
func (c *CacheManager) GetCacheStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return CacheStats{
		TotalFiles:   len(c.index),
		TotalSize:    c.calculateTotalSize(),
		MaxSize:      c.maxSize,
		UsagePercent: float64(c.calculateTotalSize()) / float64(c.maxSize) * 100,
		OldestEntry:  c.findOldestEntry(),
		NewestEntry:  c.findNewestEntry(),
	}
}

// findOldestEntry 找到最旧的缓存条目
func (c *CacheManager) findOldestEntry() *CacheEntry {
	var oldest *CacheEntry
	for _, entry := range c.index {
		if oldest == nil || entry.CreateTime.Before(oldest.CreateTime) {
			oldest = entry
		}
	}
	return oldest
}

// findNewestEntry 找到最新的缓存条目
func (c *CacheManager) findNewestEntry() *CacheEntry {
	var newest *CacheEntry
	for _, entry := range c.index {
		if newest == nil || entry.CreateTime.After(newest.CreateTime) {
			newest = entry
		}
	}
	return newest
}

// SetMaxSize 设置最大缓存大小
func (c *CacheManager) SetMaxSize(size int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.maxSize = size
}

// VerifyChecksum 验证缓存文件的完整性
func (c *CacheManager) VerifyChecksum(key string) (bool, error) {
	c.mutex.RLock()
	entry, exists := c.index[key]
	c.mutex.RUnlock()

	if !exists {
		return false, fmt.Errorf("cache entry not found for key: %s", key)
	}

	file, err := os.Open(entry.FilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, err
	}

	currentChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return currentChecksum == entry.Checksum, nil
}

// startAutoCleanup 启动自动清理定时器
func (c *CacheManager) startAutoCleanup() {
	c.cleanupCtx, c.cleanupCancel = context.WithCancel(context.Background())
	c.stopCleanup = make(chan struct{})
	c.cleanupTicker = time.NewTicker(1 * time.Hour) // 每小时检查一次

	go func() {
		defer c.cleanupTicker.Stop()

		for {
			select {
			case <-c.cleanupTicker.C:
				// 执行自动清理
				if err := c.Cleanup(); err != nil {
					// TODO: 添加日志记录
					continue
				}

				// 清理过期的临时文件
				c.cleanupTempFiles()

			case <-c.stopCleanup:
				return
			case <-c.cleanupCtx.Done():
				return
			}
		}
	}()
}

// StopAutoCleanup 停止自动清理
func (c *CacheManager) StopAutoCleanup() {
	if c.cleanupCancel != nil {
		c.cleanupCancel()
	}
	if c.stopCleanup != nil {
		close(c.stopCleanup)
	}
}

// cleanupTempFiles 清理过期的临时文件
func (c *CacheManager) cleanupTempFiles() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 扫描缓存目录，移除不在索引中的文件
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(c.cacheDir, entry.Name())

		// 跳过索引文件
		if entry.Name() == ".cache_index.json" {
			continue
		}

		// 检查文件是否在索引中
		inIndex := false
		for _, cacheEntry := range c.index {
			if cacheEntry.FilePath == filePath {
				inIndex = true
				break
			}
		}

		// 如果文件不在索引中，删除它
		if !inIndex {
			os.Remove(filePath)
		}
	}
}

// GetCacheMetrics 获取详细的缓存指标
func (c *CacheManager) GetCacheMetrics() CacheMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var totalSize int64
	var hitCount, missCount int64
	accessTimes := make([]time.Time, 0, len(c.index))
	createTimes := make([]time.Time, 0, len(c.index))

	for _, entry := range c.index {
		totalSize += entry.Size
		accessTimes = append(accessTimes, entry.AccessTime)
		createTimes = append(createTimes, entry.CreateTime)
	}

	// 计算平均访问时间
	var avgAccessTime time.Time
	if len(accessTimes) > 0 {
		var totalNano int64
		for _, t := range accessTimes {
			totalNano += t.UnixNano()
		}
		avgAccessTime = time.Unix(0, totalNano/int64(len(accessTimes)))
	}

	return CacheMetrics{
		TotalFiles:        len(c.index),
		TotalSize:         totalSize,
		MaxSize:           c.maxSize,
		UsagePercent:      float64(totalSize) / float64(c.maxSize) * 100,
		HitRate:           float64(hitCount) / float64(hitCount+missCount),
		AverageFileSize:   float64(totalSize) / float64(len(c.index)),
		AverageAccessTime: avgAccessTime,
		OldestEntry:       c.findOldestEntry(),
		NewestEntry:       c.findNewestEntry(),
		LastCleanupTime:   time.Now(), // 这里应该从实际的清理记录中获取
	}
}

// CompactCache 压缩缓存索引，移除无效条目
func (c *CacheManager) CompactCache() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	validEntries := make(map[string]*CacheEntry)

	for key, entry := range c.index {
		// 检查文件是否存在
		if _, err := os.Stat(entry.FilePath); err == nil {
			// 验证文件完整性
			if valid, err := c.verifyChecksum(entry); err == nil && valid {
				validEntries[key] = entry
			} else {
				// 文件损坏，删除
				os.Remove(entry.FilePath)
			}
		}
	}

	c.index = validEntries
	return c.saveIndex()
}

// verifyChecksum 验证单个文件的校验和
func (c *CacheManager) verifyChecksum(entry *CacheEntry) (bool, error) {
	file, err := os.Open(entry.FilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, err
	}

	currentChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return currentChecksum == entry.Checksum, nil
}

// GetEntryByPath 根据文件路径获取缓存条目
func (c *CacheManager) GetEntryByPath(filePath string) (*CacheEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, entry := range c.index {
		if entry.FilePath == filePath {
			return entry, true
		}
	}
	return nil, false
}

// UpdateAccessTime 更新条目的访问时间
func (c *CacheManager) UpdateAccessTime(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, exists := c.index[key]; exists {
		entry.AccessTime = time.Now()
	}
}

// GetLRUEntries 获取按 LRU 排序的条目列表
func (c *CacheManager) GetLRUEntries() []*CacheEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entries := make([]*CacheEntry, 0, len(c.index))
	for _, entry := range c.index {
		entries = append(entries, entry)
	}

	// 按访问时间排序 (最久未访问的在前)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AccessTime.Before(entries[j].AccessTime)
	})

	return entries
}

// PreallocateSpace 预分配缓存空间
func (c *CacheManager) PreallocateSpace(requiredSize int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	currentSize := c.calculateTotalSize()
	if currentSize+requiredSize <= c.maxSize {
		return nil // 有足够空间
	}

	// 需要清理空间
	sizeToFree := (currentSize + requiredSize) - c.maxSize
	return c.freeSpace(sizeToFree)
}

// freeSpace 释放指定大小的缓存空间
func (c *CacheManager) freeSpace(sizeToFree int64) error {
	entries := make([]*CacheEntry, 0, len(c.index))
	for _, entry := range c.index {
		entries = append(entries, entry)
	}

	// 按访问时间排序 (最久未访问的在前)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AccessTime.Before(entries[j].AccessTime)
	})

	var freedSize int64
	for _, entry := range entries {
		if freedSize >= sizeToFree {
			break
		}

		if err := os.Remove(entry.FilePath); err == nil {
			freedSize += entry.Size
			delete(c.index, entry.Key)
		}
	}

	return c.saveIndex()
}
