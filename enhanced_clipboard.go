package main

import (
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Additional clipboard formats beyond basic text (reuse existing CF_UNICODETEXT)
const (
	CF_HTML  = 49356
	CF_RTF   = 49476
	CF_HDROP = 15
)

// Enhanced clipboard format information
type ClipboardFormat struct {
	ID   uint32
	Name string
	MIME string
}

// Advanced clipboard configuration
type AdvancedClipboardConfig struct {
	DetectMultipleFormats bool
	TrackContentSize      bool
	FilterNullValues      bool
	MaxContentLength      int
	TruncateThreshold     int
}

// Null value patterns for filtering
var nullValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^null$`),
	regexp.MustCompile(`^undefined$`),
	regexp.MustCompile(`^unknown$`),
	regexp.MustCompile(`^\s*$`),
	regexp.MustCompile(`^n/a$`),
	regexp.MustCompile(`^none$`),
	regexp.MustCompile(`^empty$`),
	regexp.MustCompile(`^void$`),
	regexp.MustCompile(`^blank$`),
	regexp.MustCompile(`^na$`),
	regexp.MustCompile(`^nil$`),
	regexp.MustCompile(`^nothing$`),
	regexp.MustCompile(`^no data$`),
	regexp.MustCompile(`^no content$`),
	regexp.MustCompile(`^not available$`),
	regexp.MustCompile(`^not specified$`),
	regexp.MustCompile(`^unspecified$`),
	regexp.MustCompile(`^default$`),
	regexp.MustCompile(`^placeholder$`),
	regexp.MustCompile(`^sample$`),
}

// Supported clipboard formats
var supportedFormats = []ClipboardFormat{
	{CF_TEXT, "CF_TEXT", "text/plain"},
	{CF_UNICODETEXT, "CF_UNICODETEXT", "text/plain; charset=utf-8"},
	{CF_HTML, "CF_HTML", "text/html"},
	{CF_RTF, "CF_RTF", "application/rtf"},
	{CF_HDROP, "CF_HDROP", "application/x-file-list"},
}

// Enhanced clipboard detection
func detectClipboardFormats() []ClipboardFormat {
	var availableFormats []ClipboardFormat

	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return availableFormats
	}
	defer procCloseClipboard.Call()

	for _, format := range supportedFormats {
		ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format.ID))
		if ret != 0 {
			availableFormats = append(availableFormats, format)
		}
	}

	return availableFormats
}

// Get best available clipboard format
func getBestClipboardFormat() ClipboardFormat {
	formats := detectClipboardFormats()

	// Priority order: HTML > RTF > Unicode Text > Text
	priorities := []uint32{CF_HTML, CF_RTF, CF_UNICODETEXT, CF_TEXT}

	for _, priority := range priorities {
		for _, format := range formats {
			if format.ID == priority {
				return format
			}
		}
	}

	// Return default if nothing found
	return ClipboardFormat{CF_TEXT, "CF_TEXT", "text/plain"}
}

// Enhanced clipboard content retrieval
func getEnhancedClipboardContent() (string, ClipboardFormat, int, bool) {
	config := AdvancedClipboardConfig{
		DetectMultipleFormats: true,
		TrackContentSize:      true,
		FilterNullValues:      true,
		MaxContentLength:      globalState.Config.MaxClipboardContentLength,
		TruncateThreshold:     globalState.Config.MaxClipboardContentLength,
	}

	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return "", ClipboardFormat{}, 0, false
	}
	defer procCloseClipboard.Call()

	bestFormat := getBestClipboardFormat()
	content := ""

	switch bestFormat.ID {
	case CF_HTML:
		content = getClipboardDataByFormat(CF_HTML)
	case CF_RTF:
		content = getClipboardDataByFormat(CF_RTF)
	case CF_UNICODETEXT:
		content = getClipboardDataByFormat(CF_UNICODETEXT)
	case CF_TEXT:
		content = getClipboardDataByFormat(CF_TEXT)
	default:
		content = getClipboardContent() // Fallback to existing method
	}

	originalSize := len(content)
	truncated := false

	// Filter null values if enabled
	if config.FilterNullValues {
		if isNullValue(content) {
			return "", bestFormat, 0, false
		}
	}

	// Apply size limits
	if config.MaxContentLength > 0 && len(content) > config.MaxContentLength {
		content = content[:config.MaxContentLength]
		truncated = true
	}

	return content, bestFormat, originalSize, truncated
}

// Get clipboard data by specific format
func getClipboardDataByFormat(format uint32) string {
	ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	if ret == 0 {
		return ""
	}

	handle, _, _ := procGetClipboardData.Call(uintptr(format))
	if handle == 0 {
		return ""
	}

	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(handle)

	switch format {
	case CF_UNICODETEXT, CF_HTML, CF_RTF:
		return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(ptr))[:])
	case CF_TEXT:
		return string((*[1 << 20]byte)(unsafe.Pointer(ptr))[:])
	case CF_HDROP:
		return "[File Drop]" // Simplified representation
	default:
		return string((*[1 << 20]byte)(unsafe.Pointer(ptr))[:])
	}
}

// Check if content represents a null/empty value
func isNullValue(content string) bool {
	if content == "" {
		return true
	}

	normalizedContent := strings.ToLower(strings.TrimSpace(content))

	for _, pattern := range nullValuePatterns {
		if pattern.MatchString(normalizedContent) {
			return true
		}
	}

	return false
}

// Enhanced clipboard event creation
func createEnhancedClipboardEvent(action ClipboardAction) *ClipboardEvent {
	content, format, originalSize, truncated := getEnhancedClipboardContent()

	// Don't create event for null/empty content
	if content == "" {
		return nil
	}

	// Don't create event if content hasn't changed
	if content == globalState.LastClipboardContent {
		return nil
	}

	globalState.LastClipboardContent = content

	return &ClipboardEvent{
		Action:      action,
		Content:     content,
		ContentSize: originalSize,
		Format:      format.MIME,
		Truncated:   truncated,
		Metadata:    createEventMetadata(),
	}
}

// Monitor clipboard changes continuously
func monitorClipboardChanges() {
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for range ticker.C {
		if !globalState.Config.RecordClipboard {
			continue
		}

		content, _, _, _ := getEnhancedClipboardContent()

		// Check if clipboard content has changed
		if content != "" && content != globalState.LastClipboardContent {
			event := createEnhancedClipboardEvent(ClipboardCopy)
			if event != nil {
				logClipboardEvent(event)
			}
		}
	}
}

// Detect clipboard action based on keyboard input
func detectClipboardAction() ClipboardAction {
	// Check for common clipboard shortcuts
	if isKeyPressed(0x11) { // Ctrl key
		if isKeyPressed(0x43) { // C key
			return ClipboardCopy
		}
		if isKeyPressed(0x58) { // X key
			return ClipboardCut
		}
		if isKeyPressed(0x56) { // V key
			return ClipboardPaste
		}
	}

	return ClipboardCopy // Default action
}

// Enhanced clipboard validation
func validateClipboardContent(content string) bool {
	// Basic validation rules
	if len(content) == 0 {
		return false
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"javascript:",
		"data:text/html",
		"<script",
		"vbscript:",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerContent, pattern) {
			return false
		}
	}

	return true
}

// Get clipboard format name by ID
func getFormatName(formatID uint32) string {
	for _, format := range supportedFormats {
		if format.ID == formatID {
			return format.Name
		}
	}
	return "Unknown"
}

// Log clipboard event (placeholder for integration)
func logClipboardEvent(event *ClipboardEvent) {
	// This would integrate with the main event logging system
	// For now, this is a placeholder
}

// Enhanced clipboard processing for workflow events
func processEnhancedClipboardEvents(events *[]WorkflowEvent) {
	if !globalState.Config.RecordClipboard {
		return
	}

	// Detect recent clipboard operations
	currentContent, format, size, truncated := getEnhancedClipboardContent()

	if currentContent != "" && currentContent != globalState.LastClipboardContent {
		action := detectClipboardAction()

		event := &ClipboardEvent{
			Action:      action,
			Content:     currentContent,
			ContentSize: size,
			Format:      format.MIME,
			Truncated:   truncated,
			Metadata:    createEventMetadata(),
		}

		*events = append(*events, event)
		globalState.LastClipboardContent = currentContent
	}
}
