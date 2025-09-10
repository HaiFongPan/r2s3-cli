package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestFileBrowser_UploadFileWithSpaces 测试文件名包含空格的上传
func TestFileBrowser_UploadFileWithSpaces(t *testing.T) {
	// 创建包含空格的临时文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test file with spaces.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 创建 mock 上传器
	mockUploader := &MockFileUploader{}
	mockUploader.On("UploadFileWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 创建 FileBrowser 实例
	model := createTestFileBrowser()
	model.fileUploader = mockUploader

	// 模拟通过文本输入上传文件
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(testFile)

	// 模拟按下 Enter 键
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)

	// 验证结果
	fbModel, ok := updatedModel.(*FileBrowserModel)
	require.True(t, ok)
	
	// 应该关闭输入框
	assert.False(t, fbModel.showInput)
	assert.Equal(t, InputModeNone, fbModel.inputMode)
	
	// 应该有上传命令
	assert.NotNil(t, cmd)
	
	// 执行上传命令来触发 mock
	if cmd != nil {
		msg := cmd()
		
		// 验证上传完成消息
		uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
		assert.True(t, ok, "Expected uploadCompletedMsg")
		assert.Equal(t, "test file with spaces.txt", uploadCompletedMsg.file)
		assert.NoError(t, uploadCompletedMsg.err)
	}
	
	mockUploader.AssertExpectations(t)
}

// TestFileBrowser_TextInputWithNonExistentDirectory 测试文本输入不存在目录的处理
func TestFileBrowser_TextInputWithNonExistentDirectory(t *testing.T) {
	model := createTestFileBrowser()
	
	// 设置一个不存在的目录路径
	nonExistentPath := "/completely/nonexistent/directory/file.txt"
	
	// 模拟进入上传模式并输入不存在的路径
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(nonExistentPath)
	
	// 调用更新文件选择器的方法
	model.updateFilePickerFromTextInputSync()
	
	// 应该显示目录不存在的错误消息
	assert.Contains(t, model.statusMessage, "Directory does not exist")
	assert.Equal(t, MessageError, model.messageType)
}

// TestFileBrowser_HandleInvalidPathError 测试处理无效路径的错误消息
func TestFileBrowser_HandleInvalidPathError(t *testing.T) {
	model := createTestFileBrowser()
	
	// 设置一个不存在的文件路径
	nonExistentFile := "/completely/nonexistent/file.txt"
	
	// 模拟通过文本输入上传不存在的文件
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(nonExistentFile)

	// 模拟按下 Enter 键
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)

	// 验证结果
	fbModel, ok := updatedModel.(*FileBrowserModel)
	require.True(t, ok)
	
	// 应该关闭输入框
	assert.False(t, fbModel.showInput)
	
	// 应该显示错误消息
	assert.Contains(t, fbModel.statusMessage, "File not found")
	assert.Equal(t, MessageError, fbModel.messageType)
	
	// 不应该有上传命令（因为文件不存在）
	assert.Nil(t, cmd)
}

// TestFileBrowser_SmartDirectoryNavigation 测试智能目录导航
func TestFileBrowser_SmartDirectoryNavigation(t *testing.T) {
	model := createTestFileBrowser()
	
	// 创建临时目录结构
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	
	testFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)
	
	// 测试输入文件路径时自动更新文件选择器目录
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(testFile)
	
	// 调用更新方法
	model.updateFilePickerFromTextInputSync()
	
	// 文件选择器应该更新到文件所在的目录
	expectedDir := filepath.Dir(testFile)
	assert.Equal(t, expectedDir, model.filePicker.CurrentDirectory)
}

// TestFileBrowser_TildeExpansion 测试波浪号扩展
func TestFileBrowser_TildeExpansion(t *testing.T) {
	model := createTestFileBrowser()
	
	// 创建在用户主目录下的测试文件
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	
	// 使用相对路径（波浪号）
	tildePath := "~/test_file_for_tilde.txt"
	actualPath := filepath.Join(homeDir, "test_file_for_tilde.txt")
	
	// 创建测试文件
	err = os.WriteFile(actualPath, []byte("test content"), 0644)
	require.NoError(t, err)
	defer os.Remove(actualPath) // 清理
	
	// 创建 mock 上传器，期望接收展开后的路径
	mockUploader := &MockFileUploader{}
	mockUploader.On("UploadFileWithProgress", mock.Anything, actualPath, "test_file_for_tilde.txt", mock.Anything, mock.Anything).Return(nil)
	
	model.fileUploader = mockUploader
	
	// 模拟通过文本输入上传文件（使用波浪号路径）
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(tildePath)

	// 模拟按下 Enter 键
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)

	// 验证结果
	fbModel, ok := updatedModel.(*FileBrowserModel)
	require.True(t, ok)
	
	// 应该关闭输入框
	assert.False(t, fbModel.showInput)
	
	// 应该有上传命令
	assert.NotNil(t, cmd)
	
	// 执行上传命令来触发 mock
	if cmd != nil {
		msg := cmd()
		
		// 验证上传完成消息
		uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
		assert.True(t, ok, "Expected uploadCompletedMsg")
		assert.Equal(t, "test_file_for_tilde.txt", uploadCompletedMsg.file)
		assert.NoError(t, uploadCompletedMsg.err)
	}
	
	mockUploader.AssertExpectations(t)
}

// TestFileBrowser_UploadFileWithSingleQuotes 测试文件名包含单引号的上传
func TestFileBrowser_UploadFileWithSingleQuotes(t *testing.T) {
	// 创建包含单引号的临时文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "'Weixin Image_20250910110618_8_238.png'")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 创建 mock 上传器
	mockUploader := &MockFileUploader{}
	mockUploader.On("UploadFileWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 创建 FileBrowser 实例
	model := createTestFileBrowser()
	model.fileUploader = mockUploader

	// 模拟通过文本输入上传文件
	model.showInput = true
	model.inputMode = InputModeUpload
	model.inputComponentMode = InputComponentText
	model.textInput.SetValue(testFile)

	// 模拟按下 Enter 键
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)

	// 验证结果
	fbModel, ok := updatedModel.(*FileBrowserModel)
	require.True(t, ok)
	
	// 应该关闭输入框
	assert.False(t, fbModel.showInput)
	assert.Equal(t, InputModeNone, fbModel.inputMode)
	
	// 应该有上传命令
	assert.NotNil(t, cmd)
	
	// 执行上传命令来触发 mock
	if cmd != nil {
		msg := cmd()
		
		// 验证上传完成消息
		uploadCompletedMsg, ok := msg.(uploadCompletedMsg)
		assert.True(t, ok, "Expected uploadCompletedMsg")
		assert.Equal(t, "'Weixin Image_20250910110618_8_238.png'", uploadCompletedMsg.file)
		assert.NoError(t, uploadCompletedMsg.err)
	}
	
	mockUploader.AssertExpectations(t)
}