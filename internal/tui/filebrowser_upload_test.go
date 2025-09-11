package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/messaging"
	"github.com/HaiFongPan/r2s3-cli/internal/utils"
)

// MockFileUploader 模拟文件上传器
type MockFileUploader struct {
	mock.Mock
}

func (m *MockFileUploader) UploadFile(ctx context.Context, localPath, remotePath string, options *utils.UploadOptions) error {
	args := m.Called(ctx, localPath, remotePath, options)
	return args.Error(0)
}

func (m *MockFileUploader) UploadFileWithProgress(ctx context.Context, localPath, remotePath string, options *utils.UploadOptions, callback utils.ProgressCallback) error {
	args := m.Called(ctx, localPath, remotePath, options, callback)

	// 模拟进度回调
	if callback != nil {
		go func() {
			time.Sleep(10 * time.Millisecond)
			callback(50, 100, 50.0)
			time.Sleep(10 * time.Millisecond)
			callback(100, 100, 100.0)
		}()
	}

	return args.Error(0)
}

func (m *MockFileUploader) CheckFileExists(ctx context.Context, remotePath string) (bool, error) {
	args := m.Called(ctx, remotePath)
	return args.Bool(0), args.Error(1)
}

// TestFileBrowser_UploadFile_Success 测试成功上传文件
func TestFileBrowser_UploadFile_Success(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 创建 mock 上传器
	mockUploader := &MockFileUploader{}
	mockUploader.On("UploadFileWithProgress", mock.Anything, testFile, "test.txt", mock.Anything, mock.Anything).Return(nil)

	// 创建 FileBrowser 实例
	model := createTestFileBrowser()
	model.fileUploader = mockUploader

	// 模拟上传文件
	uploadCmd := model.uploadFile(testFile)
	require.NotNil(t, uploadCmd)

	// 执行上传命令
	msg := uploadCmd()

	// 验证结果
	uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
	require.True(t, ok, "Expected uploadCompletedMsg")
	assert.Equal(t, "test.txt", uploadCompletedMsg.file)
	assert.NoError(t, uploadCompletedMsg.err)

	mockUploader.AssertExpectations(t)
}

// TestFileBrowser_UploadFile_Error 测试上传文件失败
func TestFileBrowser_UploadFile_Error(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 创建 mock 上传器，设置返回错误
	mockUploader := &MockFileUploader{}
	expectedErr := assert.AnError
	mockUploader.On("UploadFileWithProgress", mock.Anything, testFile, "test.txt", mock.Anything, mock.Anything).Return(expectedErr)

	// 创建 FileBrowser 实例
	model := createTestFileBrowser()
	model.fileUploader = mockUploader

	// 模拟上传文件
	uploadCmd := model.uploadFile(testFile)
	require.NotNil(t, uploadCmd)

	// 执行上传命令
	msg := uploadCmd()

	// 验证结果
	uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
	require.True(t, ok, "Expected uploadCompletedMsg")
	assert.Equal(t, "test.txt", uploadCompletedMsg.file)
	assert.Equal(t, expectedErr, uploadCompletedMsg.err)

	mockUploader.AssertExpectations(t)
}

// TestFileBrowser_UploadWithProgress 测试带进度的上传
func TestFileBrowser_UploadWithProgress(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content with progress"), 0644)
	require.NoError(t, err)

	// 创建 mock 上传器
	mockUploader := &MockFileUploader{}
	mockUploader.On("UploadFileWithProgress", mock.Anything, testFile, "test.txt", mock.Anything, mock.Anything).Return(nil)

	// 创建 FileBrowser 实例
	model := createTestFileBrowser()
	model.fileUploader = mockUploader

	// 注意：没有设置program，进度回调不会实际发送消息
	// 这个测试主要验证上传逻辑本身

	// 模拟上传文件
	uploadCmd := model.uploadFile(testFile)
	require.NotNil(t, uploadCmd)

	// 执行上传命令
	msg := uploadCmd()

	// 等待进度更新
	time.Sleep(50 * time.Millisecond)

	// 验证上传完成消息
	uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
	require.True(t, ok, "Expected uploadCompletedMsg")
	assert.Equal(t, "test.txt", uploadCompletedMsg.file)
	assert.NoError(t, uploadCompletedMsg.err)

	// 注意：由于没有设置program，进度更新不会被实际发送
	// 这个测试主要验证上传完成消息

	mockUploader.AssertExpectations(t)
}

// TestFileBrowser_HandleUploadMessages 测试处理上传相关消息
func TestFileBrowser_HandleUploadMessages(t *testing.T) {
	model := createTestFileBrowser()

	// 测试上传完成消息
	t.Run("Upload Completed Success", func(t *testing.T) {
		msg := uploadCompletedMsg{file: "test.txt", err: nil}
		updatedModel, cmd := model.Update(msg)

		fbModel, ok := updatedModel.(*FileBrowserModel)
		require.True(t, ok)

		// 验证上传状态被重置
		assert.False(t, fbModel.uploading)
		assert.Empty(t, fbModel.uploadingFile)

		// 验证状态消息
		message, _, hasMessage := fbModel.messageManager.GetMessage()
		assert.True(t, hasMessage)
		assert.Contains(t, message, "uploaded successfully")

		// 应该有刷新文件列表的命令
		assert.NotNil(t, cmd)
	})

	t.Run("Upload Completed Error", func(t *testing.T) {
		msg := uploadCompletedMsg{file: "test.txt", err: assert.AnError}
		updatedModel, cmd := model.Update(msg)

		fbModel, ok := updatedModel.(*FileBrowserModel)
		require.True(t, ok)

		// 验证上传状态被重置
		assert.False(t, fbModel.uploading)
		assert.Empty(t, fbModel.uploadingFile)

		// 验证错误消息
		message, _, hasMessage := fbModel.messageManager.GetMessage()
		assert.True(t, hasMessage)
		assert.Contains(t, message, "Upload failed")

		// 不应该刷新文件列表
		assert.Nil(t, cmd)
	})
}

// TestFileBrowser_UploadKeyBinding 测试上传按键绑定
func TestFileBrowser_UploadKeyBinding(t *testing.T) {
	model := createTestFileBrowser()

	// 模拟按下 'u' 键
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}

	updatedModel, cmd := model.Update(keyMsg)

	fbModel, ok := updatedModel.(*FileBrowserModel)
	require.True(t, ok)

	// 验证进入上传输入模式
	assert.True(t, fbModel.showInput)
	assert.Equal(t, InputModeUpload, fbModel.inputMode)
	assert.Equal(t, InputComponentText, fbModel.inputComponentMode)

	// 验证输入框配置
	assert.Contains(t, fbModel.textInput.Placeholder, "file path")
	assert.True(t, fbModel.textInput.Focused())

	// 不应该有命令
	assert.Nil(t, cmd)
}

// 辅助函数和类型

// createTestFileBrowser 创建用于测试的 FileBrowser 实例
func createTestFileBrowser() *FileBrowserModel {
	cfg := &config.Config{
		Upload: config.UploadConfig{
			DefaultOverwrite: false,
			DefaultPublic:    false,
		},
	}

	// 创建一个基本的模型结构
	model := &FileBrowserModel{
		config:     cfg,
		bucketName: "test-bucket",
		prefix:     "",

		// 设置基本状态
		windowWidth:  80,
		windowHeight: 24,

		// 上传状态
		uploading:     false,
		uploadingFile: "",

		// 输入状态
		showInput:          false,
		inputMode:          InputModeNone,
		inputComponentMode: InputComponentText,

		// 消息管理器
		messageManager: messaging.NewStatusManager(),

		// 键映射
		keyMap: DefaultKeyMap(),
	}

	// 初始化输入组件
	model.textInput = textinput.New()

	return model
}

// mockProgram 模拟 bubbletea 程序，用于测试消息发送
type mockProgram struct {
	sentMessages *[]tea.Msg
}

func (p *mockProgram) Send(msg tea.Msg) {
	*p.sentMessages = append(*p.sentMessages, msg)
}

// 实现 *tea.Program 需要的其他方法（仅为测试目的）
func (p *mockProgram) Start() error                   { return nil }
func (p *mockProgram) StartReturningModel() tea.Model { return nil }
func (p *mockProgram) Wait() error                    { return nil }
func (p *mockProgram) Kill()                          {}
func (p *mockProgram) Quit()                          {}
func (p *mockProgram) ReleaseTerminal() error         { return nil }
func (p *mockProgram) RestoreTerminal() error         { return nil }
