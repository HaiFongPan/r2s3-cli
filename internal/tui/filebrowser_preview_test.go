package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// Test 'p' opens modal preview safely
func TestFileBrowser_ToggleImagePreview_KeyBinding(t *testing.T) {
	model := createTestFileBrowser()

	// Sample image entry
	model.files = []FileItem{{
		Key:         "photos/cat.png",
		Size:        1024,
		ContentType: "image/png",
		Category:    "image",
	}}
	model.cursor = 0

	// Press 'p'
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	fb, ok := updated.(*FileBrowserModel)
	if !ok {
		t.Fatalf("updated model is not *FileBrowserModel")
	}

	assert.True(t, fb.showingPreview)
	assert.NotNil(t, fb.previewModal)
	assert.NotNil(t, cmd)
	if fb.previewModal != nil {
		assert.False(t, fb.previewModal.forceReload)
	}
}

// Test 'P' forces modal preview reload
func TestFileBrowser_ForceImagePreview_KeyBinding(t *testing.T) {
	model := createTestFileBrowser()

	model.files = []FileItem{{
		Key:         "photos/cat.png",
		Size:        1024,
		ContentType: "image/png",
		Category:    "image",
	}}
	model.cursor = 0

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	fb, ok := updated.(*FileBrowserModel)
	if !ok {
		t.Fatalf("updated model is not *FileBrowserModel")
	}

	assert.True(t, fb.showingPreview)
	assert.NotNil(t, fb.previewModal)
	assert.NotNil(t, cmd)
	if fb.previewModal != nil {
		assert.True(t, fb.previewModal.forceReload)
	}
}
