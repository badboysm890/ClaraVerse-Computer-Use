package main

import (
	"log"
	"math"
	"sync"
	"time"
)

// SelectionMethod represents how text was selected
type SelectionMethod string

const (
	SelectionMouseDrag        SelectionMethod = "MouseDrag"
	SelectionDoubleClick      SelectionMethod = "DoubleClick"      // Word selection
	SelectionTripleClick      SelectionMethod = "TripleClick"      // Line/paragraph selection
	SelectionKeyboardShortcut SelectionMethod = "KeyboardShortcut" // Ctrl+A, Shift+arrows, etc.
	SelectionContextMenu      SelectionMethod = "ContextMenu"
)

// TextSelectionEvent represents a text selection event
type TextSelectionEvent struct {
	SelectedText    string          `json:"selected_text"`
	StartPosition   Position        `json:"start_position"`
	EndPosition     Position        `json:"end_position"`
	SelectionMethod SelectionMethod `json:"selection_method"`
	SelectionLength uint32          `json:"selection_length"`
	Metadata        EventMetadata   `json:"metadata"`
}

// TextSelectionTracker tracks text selection events
type TextSelectionTracker struct {
	IsSelecting        bool
	SelectionStartPos  Position
	SelectionStartTime time.Time
	LastMousePos       Position
	LastClickTime      time.Time
	LastClickPos       Position
	ClickCount         int
	EventCallback      func(TextSelectionEvent)
	Mutex              sync.RWMutex
}

// NewTextSelectionTracker creates a new text selection tracker
func NewTextSelectionTracker(callback func(TextSelectionEvent)) *TextSelectionTracker {
	return &TextSelectionTracker{
		EventCallback: callback,
		ClickCount:    0,
	}
}

// HandleMouseDown processes mouse down events for selection tracking
func (tst *TextSelectionTracker) HandleMouseDown(position Position, button MouseButton) {
	if button != MouseButtonLeft {
		return
	}

	tst.Mutex.Lock()
	defer tst.Mutex.Unlock()

	now := time.Now()

	// Check for double/triple click
	if tst.isCloseClick(position, now) {
		tst.ClickCount++
	} else {
		tst.ClickCount = 1
	}

	tst.LastClickTime = now
	tst.LastClickPos = position

	// Start potential selection
	tst.IsSelecting = true
	tst.SelectionStartPos = position
	tst.SelectionStartTime = now
	tst.LastMousePos = position
}

// HandleMouseMove processes mouse move events during selection
func (tst *TextSelectionTracker) HandleMouseMove(position Position) {
	tst.Mutex.RLock()
	isSelecting := tst.IsSelecting
	tst.Mutex.RUnlock()

	if !isSelecting {
		return
	}

	tst.Mutex.Lock()
	defer tst.Mutex.Unlock()

	tst.LastMousePos = position
}

// HandleMouseUp processes mouse up events to complete selection
func (tst *TextSelectionTracker) HandleMouseUp(position Position, button MouseButton) {
	if button != MouseButtonLeft {
		return
	}

	tst.Mutex.Lock()
	defer tst.Mutex.Unlock()

	if !tst.IsSelecting {
		return
	}

	tst.IsSelecting = false

	// Check if this was actually a selection (mouse moved enough)
	distance := tst.calculateDistance(tst.SelectionStartPos, position)
	minSelectionDistance := 5.0 // pixels

	if distance < minSelectionDistance && tst.ClickCount <= 1 {
		return // Just a click, not a selection
	}

	// Determine selection method
	var method SelectionMethod
	switch tst.ClickCount {
	case 2:
		method = SelectionDoubleClick
	case 3:
		method = SelectionTripleClick
	default:
		if distance >= minSelectionDistance {
			method = SelectionMouseDrag
		} else {
			return // No significant selection
		}
	}

	// Get selected text from clipboard or UI automation
	selectedText := tst.getSelectedText()
	if selectedText == "" {
		return // No text was actually selected
	}

	// Create selection event
	event := TextSelectionEvent{
		SelectedText:    selectedText,
		StartPosition:   tst.SelectionStartPos,
		EndPosition:     position,
		SelectionMethod: method,
		SelectionLength: uint32(len(selectedText)),
		Metadata:        createEventMetadata(),
	}

	// Emit the event
	if tst.EventCallback != nil {
		go tst.EventCallback(event)
	}

	log.Printf("Text selection detected: '%s' (%s, %d chars)",
		truncateString(selectedText, 50), method, len(selectedText))
}

// HandleKeyboardShortcut processes keyboard shortcuts that might indicate text selection
func (tst *TextSelectionTracker) HandleKeyboardShortcut(combination string) {
	if !tst.isSelectionHotkey(combination) {
		return
	}

	// Wait a moment for the selection to complete
	time.AfterFunc(100*time.Millisecond, func() {
		selectedText := tst.getSelectedText()
		if selectedText == "" {
			return
		}

		// Get current cursor position (approximate)
		currentPos := getMousePosition()

		event := TextSelectionEvent{
			SelectedText:    selectedText,
			StartPosition:   currentPos,
			EndPosition:     currentPos,
			SelectionMethod: SelectionKeyboardShortcut,
			SelectionLength: uint32(len(selectedText)),
			Metadata:        createEventMetadata(),
		}

		if tst.EventCallback != nil {
			go tst.EventCallback(event)
		}

		log.Printf("Keyboard text selection detected: '%s' (%s)",
			truncateString(selectedText, 50), combination)
	})
}

// Helper methods
func (tst *TextSelectionTracker) isCloseClick(position Position, clickTime time.Time) bool {
	maxClickDistance := 5.0 // pixels
	maxClickInterval := 500 * time.Millisecond

	if time.Since(tst.LastClickTime) > maxClickInterval {
		return false
	}

	distance := tst.calculateDistance(tst.LastClickPos, position)
	return distance <= maxClickDistance
}

func (tst *TextSelectionTracker) calculateDistance(p1, p2 Position) float64 {
	dx := float64(p2.X - p1.X)
	dy := float64(p2.Y - p1.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (tst *TextSelectionTracker) getSelectedText() string {
	// Try to get selected text from clipboard
	// This is a simple approach - in a real implementation you'd want
	// to use UI Automation or other APIs to get the selected text directly

	// Store original clipboard content
	originalClipboard := getClipboardContent()

	// Simulate Ctrl+C to copy selection
	// Note: This is a simplified approach. A production implementation
	// would use proper UI Automation APIs

	// For now, return empty string - this would need proper implementation
	// using Windows UI Automation or similar APIs

	// Restore original clipboard content if we modified it
	_ = originalClipboard

	return "" // Placeholder - would need proper UI Automation implementation
}

func (tst *TextSelectionTracker) isSelectionHotkey(combination string) bool {
	selectionHotkeys := []string{
		"Ctrl+A",                              // Select all
		"Ctrl+Shift+Left", "Ctrl+Shift+Right", // Word selection
		"Shift+Left", "Shift+Right", // Character selection
		"Shift+Up", "Shift+Down", // Line selection
		"Shift+Home", "Shift+End", // Line start/end selection
		"Ctrl+Shift+Home", "Ctrl+Shift+End", // Document selection
	}

	for _, hotkey := range selectionHotkeys {
		if combination == hotkey {
			return true
		}
	}

	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
