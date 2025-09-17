package messaging

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/tui/theme"
)

// MessageType represents different message types for status display
type MessageType int

// Message type constants
const (
	MessageInfo MessageType = iota
	MessageSuccess
	MessageWarning
	MessageError
)

// StatusManager manages status messages and their display
type StatusManager interface {
	SetMessage(message string, msgType MessageType)
	ClearMessage()
	GetMessage() (string, MessageType, bool)
	RenderMessage() string
	HasMessage() bool
}

// StatusManagerImpl implements the StatusManager interface
type StatusManagerImpl struct {
	statusMessage string
	messageType   MessageType
	messageTimer  time.Time
}

// NewStatusManager creates a new status manager instance
func NewStatusManager() StatusManager {
	return &StatusManagerImpl{
		statusMessage: "",
		messageType:   MessageInfo,
	}
}

// SetMessage sets a status message with type
func (sm *StatusManagerImpl) SetMessage(message string, msgType MessageType) {
	sm.statusMessage = message
	sm.messageType = msgType
	sm.messageTimer = time.Now()

	// Debug logging
	logrus.Debugf("StatusManager: setMessage called with message='%s', type=%d", message, msgType)
}

// ClearMessage clears the status message
func (sm *StatusManagerImpl) ClearMessage() {
	sm.statusMessage = ""
	logrus.Debugf("StatusManager: message cleared")
}

// GetMessage returns the current message, type, and whether a message exists
func (sm *StatusManagerImpl) GetMessage() (string, MessageType, bool) {
	hasMessage := sm.statusMessage != ""
	return sm.statusMessage, sm.messageType, hasMessage
}

// HasMessage returns whether there is currently a status message
func (sm *StatusManagerImpl) HasMessage() bool {
	return sm.statusMessage != ""
}

// RenderMessage renders the current status message with appropriate styling
func (sm *StatusManagerImpl) RenderMessage() string {
	if !sm.HasMessage() {
		return ""
	}

	// Get message color and icon from theme
	messageColor := theme.GetMessageColor(int(sm.messageType))
	messageIcon := theme.GetMessageIcon(int(sm.messageType))

	// Create styled message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(messageColor)).
		Bold(true)

	return messageStyle.Render(fmt.Sprintf("%s %s", messageIcon, sm.statusMessage))
}
