package utils

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
)

// MockS3Client 用于模拟 S3 客户端
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	// 模拟读取 Body 内容以触发进度回调
	if params.Body != nil {
		buffer := make([]byte, 1024)
		for {
			n, err := params.Body.Read(buffer)
			if err == io.EOF || (err != nil && n == 0) {
				break
			}
			if err != nil && err != io.EOF {
				break
			}
		}
	}
	
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

// MockR2Client 用于模拟 R2 客户端
type MockR2Client struct {
	mock.Mock
	s3Client *MockS3Client
}

func (m *MockR2Client) GetS3Client() interface{} {
	return m.s3Client
}

// 测试用例开始

func TestFileUploader_UploadFile_Success(t *testing.T) {
	// 准备测试数据
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望
	// 先检查文件是否存在（返回不存在）
	mockS3Client.On("HeadObject", mock.Anything, mock.Anything).Return((*s3.HeadObjectOutput)(nil), &MockNotFoundError{})
	// 然后执行上传
	mockS3Client.On("PutObject", mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)
	
	// 创建配置
	cfg := &config.Config{
		Upload: config.UploadConfig{
			DefaultOverwrite: false,
			DefaultPublic:    false,
			DefaultCompress:  "",
		},
	}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 执行上传
	ctx := context.Background()
	options := &UploadOptions{
		Overwrite:    false,
		PublicAccess: false,
	}
	
	err = uploader.UploadFile(ctx, testFile, "remote/test.txt", options)
	
	// 验证结果
	assert.NoError(t, err)
	mockS3Client.AssertExpectations(t)
}

func TestFileUploader_UploadFile_FileNotExists(t *testing.T) {
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 执行上传（文件不存在）
	ctx := context.Background()
	options := &UploadOptions{}
	
	err := uploader.UploadFile(ctx, "/nonexistent/file.txt", "remote/test.txt", options)
	
	// 验证结果 - 应该返回错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open file failed")
}

func TestFileUploader_CheckFileExists_FileExists(t *testing.T) {
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望 - 文件存在
	mockS3Client.On("HeadObject", mock.Anything, mock.Anything).Return(&s3.HeadObjectOutput{}, nil)
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 检查文件是否存在
	ctx := context.Background()
	exists, err := uploader.CheckFileExists(ctx, "remote/test.txt")
	
	// 验证结果
	assert.NoError(t, err)
	assert.True(t, exists)
	mockS3Client.AssertExpectations(t)
}

func TestFileUploader_CheckFileExists_FileNotExists(t *testing.T) {
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望 - 文件不存在（404 错误）
	mockS3Client.On("HeadObject", mock.Anything, mock.Anything).Return((*s3.HeadObjectOutput)(nil), &MockNotFoundError{})
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 检查文件是否存在
	ctx := context.Background()
	exists, err := uploader.CheckFileExists(ctx, "remote/test.txt")
	
	// 验证结果
	assert.NoError(t, err)
	assert.False(t, exists)
	mockS3Client.AssertExpectations(t)
}

func TestFileUploader_UploadFileWithProgress_Success(t *testing.T) {
	// 准备测试数据
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World! This is a longer content to test progress callback."
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望
	// 先检查文件是否存在（返回不存在）
	mockS3Client.On("HeadObject", mock.Anything, mock.Anything).Return((*s3.HeadObjectOutput)(nil), &MockNotFoundError{})
	// 然后执行上传
	mockS3Client.On("PutObject", mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 进度回调变量
	var progressCalls []float64
	progressCallback := func(uploaded, total int64, percentage float64) {
		progressCalls = append(progressCalls, percentage)
	}
	
	// 执行上传
	ctx := context.Background()
	options := &UploadOptions{}
	
	err = uploader.UploadFileWithProgress(ctx, testFile, "remote/test.txt", options, progressCallback)
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotEmpty(t, progressCalls, "Progress callback should have been called")
	assert.Equal(t, 100.0, progressCalls[len(progressCalls)-1], "Final progress should be 100%")
	mockS3Client.AssertExpectations(t)
}

func TestFileUploader_UploadFile_WithOverwrite(t *testing.T) {
	// 准备测试数据
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望
	// 由于设置了 overwrite=true，不会检查文件是否存在，直接执行上传
	mockS3Client.On("PutObject", mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 执行上传（设置覆盖选项）
	ctx := context.Background()
	options := &UploadOptions{
		Overwrite: true,
	}
	
	err = uploader.UploadFile(ctx, testFile, "remote/test.txt", options)
	
	// 验证结果
	assert.NoError(t, err)
	mockS3Client.AssertExpectations(t)
}

func TestFileUploader_UploadFile_FileExistsNoOverwrite(t *testing.T) {
	// 准备测试数据
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// 创建 mock 客户端
	mockS3Client := &MockS3Client{}
	mockR2Client := &MockR2Client{s3Client: mockS3Client}
	
	// 设置 mock 期望 - 文件存在
	mockS3Client.On("HeadObject", mock.Anything, mock.Anything).Return(&s3.HeadObjectOutput{}, nil)
	// 不应该调用 PutObject，因为文件已存在且未设置覆盖
	
	// 创建配置
	cfg := &config.Config{}
	
	// 创建上传器
	uploader := NewFileUploader(mockR2Client, cfg, "test-bucket")
	
	// 执行上传（不设置覆盖选项）
	ctx := context.Background()
	options := &UploadOptions{
		Overwrite: false,
	}
	
	err = uploader.UploadFile(ctx, testFile, "remote/test.txt", options)
	
	// 验证结果 - 应该返回错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file conflict failed")
	mockS3Client.AssertExpectations(t)
}

// Mock 错误类型用于模拟 404 错误
type MockNotFoundError struct{}

func (e *MockNotFoundError) Error() string {
	return "NotFound: The specified key does not exist."
}

