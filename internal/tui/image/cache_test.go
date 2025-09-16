package image

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheManager(t *testing.T) {
	tempDir := t.TempDir()
	maxSize := int64(100 * 1024 * 1024) // 100MB

	cm := NewCacheManager(tempDir, maxSize)
	defer cm.StopAutoCleanup()

	assert.Equal(t, tempDir, cm.cacheDir)
	assert.Equal(t, maxSize, cm.maxSize)
	assert.NotNil(t, cm.index)
	assert.Equal(t, CleanupPolicyLRU, cm.cleanupPolicy)

	// 验证目录已创建
	_, err := os.Stat(tempDir)
	assert.NoError(t, err)
}

func TestCacheManager_PutAndGet(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 创建测试文件
	testContent := []byte("test image content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// 测试 Put
	key := "test-image"
	cachedPath, err := cm.Put(key, testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, cachedPath)

	// 验证缓存文件存在
	_, err = os.Stat(cachedPath)
	assert.NoError(t, err)

	// 测试 Get
	retrievedPath, cacheHit, err := cm.Get(key)
	require.NoError(t, err)
	assert.True(t, cacheHit)
	assert.Equal(t, cachedPath, retrievedPath)

	// 验证内容相同
	cachedContent, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, cachedContent)
}

func TestCacheManager_GetNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	path, cacheHit, err := cm.Get("non-existent")
	assert.NoError(t, err)
	assert.False(t, cacheHit)
	assert.Empty(t, path)
}

func TestCacheManager_Delete(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 添加缓存条目
	testContent := []byte("test content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	key := "test-key"
	cachedPath, err := cm.Put(key, testFile)
	require.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(cachedPath)
	assert.NoError(t, err)

	// 删除
	err = cm.Delete(key)
	assert.NoError(t, err)

	// 验证文件不存在
	_, err = os.Stat(cachedPath)
	assert.True(t, os.IsNotExist(err))

	// 验证缓存中也不存在
	_, cacheHit, err := cm.Get(key)
	assert.NoError(t, err)
	assert.False(t, cacheHit)
}

func TestCacheManager_GetSize(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 初始大小应为0
	assert.Equal(t, int64(0), cm.GetSize())

	// 添加一些文件
	testContent1 := []byte("test content 1")
	testFile1 := filepath.Join(tempDir, "test1.jpg")
	err := os.WriteFile(testFile1, testContent1, 0644)
	require.NoError(t, err)

	_, err = cm.Put("key1", testFile1)
	require.NoError(t, err)

	testContent2 := []byte("test content 2 - longer")
	testFile2 := filepath.Join(tempDir, "test2.jpg")
	err = os.WriteFile(testFile2, testContent2, 0644)
	require.NoError(t, err)

	_, err = cm.Put("key2", testFile2)
	require.NoError(t, err)

	expectedSize := int64(len(testContent1) + len(testContent2))
	assert.Equal(t, expectedSize, cm.GetSize())
}

func TestCacheManager_LRUCleanup(t *testing.T) {
	tempDir := t.TempDir()
	maxSize := int64(50) // 很小的缓存大小
	cm := NewCacheManager(tempDir, maxSize)
	defer cm.StopAutoCleanup()

	// 创建多个测试文件
	files := []struct {
		key     string
		content string
	}{
		{"file1", "content1_twelve_bytes"}, // 19 bytes
		{"file2", "content2_thirteen_byt"}, // 20 bytes
		{"file3", "content3_fourteen_byt"}, // 20 bytes
	}

	for i, file := range files {
		testFile := filepath.Join(tempDir, file.key+".txt")
		err := os.WriteFile(testFile, []byte(file.content), 0644)
		require.NoError(t, err)

		_, err = cm.Put(file.key, testFile)
		require.NoError(t, err)

		// 稍微延迟以确保访问时间不同
		if i < len(files)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 手动触发清理
	err := cm.Cleanup()
	assert.NoError(t, err)

	// 检查大小是否在限制内
	assert.LessOrEqual(t, cm.GetSize(), maxSize)
}

func TestCacheManager_VerifyChecksum(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 创建测试文件
	testContent := []byte("test content for checksum")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// 添加到缓存
	key := "test-checksum"
	_, err = cm.Put(key, testFile)
	require.NoError(t, err)

	// 验证校验和
	valid, err := cm.VerifyChecksum(key)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestCacheManager_VerifyChecksumCorrupted(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 创建测试文件
	testContent := []byte("original content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// 添加到缓存
	key := "test-corrupted"
	cachedPath, err := cm.Put(key, testFile)
	require.NoError(t, err)

	// 修改缓存文件内容（模拟损坏）
	corruptedContent := []byte("corrupted content")
	err = os.WriteFile(cachedPath, corruptedContent, 0644)
	require.NoError(t, err)

	// 验证校验和应该失败
	valid, err := cm.VerifyChecksum(key)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestCacheManager_GetCacheStats(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 初始统计
	stats := cm.GetCacheStats()
	assert.Equal(t, 0, stats.TotalFiles)
	assert.Equal(t, int64(0), stats.TotalSize)

	// 添加一些文件
	testContent := []byte("test content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	_, err = cm.Put("test-key", testFile)
	require.NoError(t, err)

	// 检查更新后的统计
	stats = cm.GetCacheStats()
	assert.Equal(t, 1, stats.TotalFiles)
	assert.Equal(t, int64(len(testContent)), stats.TotalSize)
	assert.NotNil(t, stats.OldestEntry)
	assert.NotNil(t, stats.NewestEntry)
}

func TestCacheManager_CompactCache(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 添加文件到缓存
	testContent := []byte("test content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	cachedPath, err := cm.Put("test-key", testFile)
	require.NoError(t, err)

	// 验证文件存在
	stats := cm.GetCacheStats()
	assert.Equal(t, 1, stats.TotalFiles)

	// 手动删除缓存文件（模拟文件丢失）
	err = os.Remove(cachedPath)
	require.NoError(t, err)

	// 压缩缓存
	err = cm.CompactCache()
	assert.NoError(t, err)

	// 验证无效条目已被移除
	stats = cm.GetCacheStats()
	assert.Equal(t, 0, stats.TotalFiles)
}

func TestCacheManager_PreallocateSpace(t *testing.T) {
	tempDir := t.TempDir()
	maxSize := int64(100) // 100字节的小缓存
	cm := NewCacheManager(tempDir, maxSize)
	defer cm.StopAutoCleanup()

	// 测试有足够空间的情况
	err := cm.PreallocateSpace(50)
	assert.NoError(t, err)

	// 添加一些数据
	testContent := make([]byte, 60) // 60字节
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}
	testFile := filepath.Join(tempDir, "large.jpg")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	_, err = cm.Put("large-file", testFile)
	require.NoError(t, err)

	// 现在尝试预分配更多空间（应该触发清理）
	err = cm.PreallocateSpace(50)
	assert.NoError(t, err)

	// 验证缓存大小在限制内
	assert.LessOrEqual(t, cm.GetSize()+50, maxSize)
}

func TestCacheManager_UpdateAccessTime(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 添加文件到缓存
	testContent := []byte("test content")
	testFile := filepath.Join(tempDir, "test.jpg")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	key := "test-key"
	_, err = cm.Put(key, testFile)
	require.NoError(t, err)

	// 获取初始访问时间
	entry := cm.index[key]
	initialAccessTime := entry.AccessTime

	// 等待一段时间
	time.Sleep(10 * time.Millisecond)

	// 更新访问时间
	cm.UpdateAccessTime(key)

	// 验证访问时间已更新
	updatedEntry := cm.index[key]
	assert.True(t, updatedEntry.AccessTime.After(initialAccessTime))
}

func TestCacheManager_GetLRUEntries(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCacheManager(tempDir, 10*1024*1024)
	defer cm.StopAutoCleanup()

	// 添加多个文件，每个都有不同的访问时间
	files := []string{"file1", "file2", "file3"}
	for i, fileName := range files {
		testContent := []byte("content " + fileName)
		testFile := filepath.Join(tempDir, fileName+".jpg")
		err := os.WriteFile(testFile, testContent, 0644)
		require.NoError(t, err)

		_, err = cm.Put(fileName, testFile)
		require.NoError(t, err)

		// 延迟以确保不同的访问时间
		if i < len(files)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 获取LRU排序的条目
	entries := cm.GetLRUEntries()
	assert.Equal(t, len(files), len(entries))

	// 验证按访问时间排序（最旧的在前）
	for i := 1; i < len(entries); i++ {
		assert.True(t, entries[i-1].AccessTime.Before(entries[i].AccessTime) ||
			entries[i-1].AccessTime.Equal(entries[i].AccessTime))
	}
}
