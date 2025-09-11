package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/messaging"
)

// TestFileBrowser_DeleteStatusIndication 测试删除状态指示
func TestFileBrowser_DeleteStatusIndication(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{
		R2: config.R2Config{
			AccountID:       "test-account",
			AccessKeyID:     "test-key",
			AccessKeySecret: "test-secret",
		},
	}

	// 创建 R2 客户端（mock）
	client, err := r2.NewClient(&cfg.R2)
	require.NoError(t, err)

	// 创建 FileBrowser 模型
	model := NewFileBrowserModel(client, cfg, "test-bucket", "")

	// 添加一个测试文件到模型中
	model.files = []FileItem{
		{
			Key:  "test-file.txt",
			Size: 1024,
		},
	}
	model.cursor = 0

	// 测试按下删除键
	deleteKeyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	updatedModel, cmd := model.Update(deleteKeyMsg)
	fbModel := updatedModel.(*FileBrowserModel)

	// 验证删除确认状态
	assert.True(t, fbModel.confirmDelete)
	assert.Equal(t, "test-file.txt", fbModel.deleteTarget)
	assert.Nil(t, cmd)

	// 测试确认删除
	confirmKeyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updatedModel2, cmd2 := fbModel.Update(confirmKeyMsg)
	fbModel2 := updatedModel2.(*FileBrowserModel)

	// 验证删除状态
	assert.False(t, fbModel2.confirmDelete)
	assert.True(t, fbModel2.deleting)
	assert.Equal(t, "test-file.txt", fbModel2.deletingFile)
	message2, msgType2, hasMessage2 := fbModel2.messageManager.GetMessage()
	assert.True(t, hasMessage2)
	assert.Contains(t, message2, "Deleting test-file.txt...")
	assert.Equal(t, messaging.MessageWarning, msgType2)
	assert.NotNil(t, cmd2)

	// 模拟删除完成（成功）
	deleteCompletedMsg := deleteCompletedMsg{err: nil}
	updatedModel3, cmd3 := fbModel2.Update(deleteCompletedMsg)
	fbModel3 := updatedModel3.(*FileBrowserModel)

	// 验证删除完成后的状态
	assert.False(t, fbModel3.deleting)
	assert.Equal(t, "", fbModel3.deletingFile)
	message3, msgType3, hasMessage3 := fbModel3.messageManager.GetMessage()
	assert.True(t, hasMessage3)
	assert.Contains(t, message3, "deleted successfully")
	assert.Equal(t, messaging.MessageSuccess, msgType3)
	assert.NotNil(t, cmd3) // 应该有重新加载文件的命令
}

// TestFileBrowser_DeleteStatusBlocking 测试删除期间的操作阻塞
func TestFileBrowser_DeleteStatusBlocking(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{
		R2: config.R2Config{
			AccountID:       "test-account",
			AccessKeyID:     "test-key",
			AccessKeySecret: "test-secret",
		},
	}

	// 创建 R2 客户端（mock）
	client, err := r2.NewClient(&cfg.R2)
	require.NoError(t, err)

	// 创建 FileBrowser 模型
	model := NewFileBrowserModel(client, cfg, "test-bucket", "")

	// 设置删除状态
	model.deleting = true
	model.deletingFile = "test.txt"

	testCases := []struct {
		name string
		key  rune
	}{
		{"Upload", 'u'},
		{"Search", '/'},
		{"Download", 'd'},
		{"Preview", 'p'},
		{"Delete", 'x'},
		{"ChangeBucket", 'b'},
		{"Refresh", 'r'},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}}
			updatedModel, cmd := model.Update(keyMsg)
			fbModel := updatedModel.(*FileBrowserModel)

			// 验证操作被阻塞
			assert.True(t, fbModel.deleting, "Delete state should remain true")
			assert.Equal(t, "test.txt", fbModel.deletingFile, "Deleting file should remain unchanged")
			assert.Nil(t, cmd, "No command should be returned when blocked")
		})
	}
}
