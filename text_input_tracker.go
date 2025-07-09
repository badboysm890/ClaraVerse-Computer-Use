package main

import (
	"log"
	"strings"
	"sync"
	"time"
)

// TextInputMethod represents how text was input
type TextInputMethod string

const (
	TextInputTyped      TextInputMethod = "Typed"
	TextInputPasted     TextInputMethod = "Pasted"
	TextInputAutoFilled TextInputMethod = "AutoFilled"
	TextInputSuggestion TextInputMethod = "Suggestion"
	TextInputMixed      TextInputMethod = "Mixed"
)

// TextInputCompletedEvent represents a completed text input session
type TextInputCompletedEvent struct {
	TextValue        string          `json:"text_value"`
	FieldName        string          `json:"field_name,omitempty"`
	FieldType        string          `json:"field_type"`
	InputMethod      TextInputMethod `json:"input_method"`
	TypingDurationMs uint64          `json:"typing_duration_ms"`
	KeystrokeCount   uint32          `json:"keystroke_count"`
	Metadata         EventMetadata   `json:"metadata"`
}

// TextInputTracker tracks text input sessions to generate completion events
type TextInputTracker struct {
	Element         *UIElement
	StartTime       time.Time
	LastKeystroke   time.Time
	KeystrokeCount  uint32
	InitialText     string
	CurrentText     string
	InputMethod     TextInputMethod
	CompletionTimer *time.Timer
	Mutex           sync.RWMutex
}

// TextInputManager manages multiple text input sessions
type TextInputManager struct {
	ActiveInputs      map[string]*TextInputTracker
	CompletionTimeout time.Duration
	EventCallback     func(TextInputCompletedEvent)
	Mutex             sync.RWMutex
}

// NewTextInputManager creates a new text input manager
func NewTextInputManager(completionTimeout time.Duration, callback func(TextInputCompletedEvent)) *TextInputManager {
	return &TextInputManager{
		ActiveInputs:      make(map[string]*TextInputTracker),
		CompletionTimeout: completionTimeout,
		EventCallback:     callback,
	}
}

// StartTextInput begins tracking a new text input session
func (tim *TextInputManager) StartTextInput(element *UIElement) {
	if element == nil {
		return
	}

	elementKey := tim.getElementKey(element)

	tim.Mutex.Lock()
	defer tim.Mutex.Unlock()

	// Complete any existing session for this element
	if existing, exists := tim.ActiveInputs[elementKey]; exists {
		tim.completeTextInputInternal(existing, "focus_change")
	}

	// Start new session
	tracker := &TextInputTracker{
		Element:        element,
		StartTime:      time.Now(),
		LastKeystroke:  time.Now(),
		KeystrokeCount: 0,
		InitialText:    tim.getCurrentText(element),
		CurrentText:    tim.getCurrentText(element),
		InputMethod:    TextInputTyped,
	}

	tracker.CompletionTimer = time.AfterFunc(tim.CompletionTimeout, func() {
		tim.CompleteTextInput(elementKey, "timeout")
	})

	tim.ActiveInputs[elementKey] = tracker
}

// HandleKeystroke processes a keystroke for text input tracking
func (tim *TextInputManager) HandleKeystroke(keyCode uint32, char string) {
	tim.Mutex.Lock()
	defer tim.Mutex.Unlock()

	// Find the currently focused text input
	for _, tracker := range tim.ActiveInputs {
		if tim.isElementFocused(tracker.Element) {
			tracker.Mutex.Lock()
			tracker.LastKeystroke = time.Now()
			tracker.KeystrokeCount++

			// Update input method based on typing pattern
			if tracker.KeystrokeCount == 1 {
				tracker.InputMethod = TextInputTyped
			} else if tim.isLikelyPasted(keyCode, char, tracker) {
				tracker.InputMethod = TextInputPasted
			} else if tracker.InputMethod == TextInputTyped && tim.isLikelySuggestion(keyCode, char, tracker) {
				tracker.InputMethod = TextInputSuggestion
			}

			// Reset completion timer
			if tracker.CompletionTimer != nil {
				tracker.CompletionTimer.Stop()
			}
			tracker.CompletionTimer = time.AfterFunc(tim.CompletionTimeout, func() {
				tim.CompleteTextInput(tim.getElementKey(tracker.Element), "timeout")
			})

			tracker.Mutex.Unlock()
			break
		}
	}
}

// CompleteTextInput finishes a text input session
func (tim *TextInputManager) CompleteTextInput(elementKey string, reason string) {
	tim.Mutex.Lock()
	defer tim.Mutex.Unlock()

	if tracker, exists := tim.ActiveInputs[elementKey]; exists {
		tim.completeTextInputInternal(tracker, reason)
		delete(tim.ActiveInputs, elementKey)
	}
}

// CompleteAllActiveInputs completes all active text input sessions
func (tim *TextInputManager) CompleteAllActiveInputs() {
	tim.Mutex.Lock()
	defer tim.Mutex.Unlock()

	for elementKey, tracker := range tim.ActiveInputs {
		tim.completeTextInputInternal(tracker, "session_end")
		delete(tim.ActiveInputs, elementKey)
	}
}

// Internal method to complete text input (must be called with mutex held)
func (tim *TextInputManager) completeTextInputInternal(tracker *TextInputTracker, reason string) {
	if tracker == nil {
		return
	}

	tracker.Mutex.Lock()
	defer tracker.Mutex.Unlock()

	// Stop the completion timer
	if tracker.CompletionTimer != nil {
		tracker.CompletionTimer.Stop()
	}

	// Get final text value
	finalText := tim.getCurrentText(tracker.Element)

	// Only emit event if text actually changed
	if finalText != tracker.InitialText && strings.TrimSpace(finalText) != "" {
		duration := time.Since(tracker.StartTime).Milliseconds()

		event := TextInputCompletedEvent{
			TextValue:        finalText,
			FieldName:        tim.getFieldName(tracker.Element),
			FieldType:        tim.getFieldType(tracker.Element),
			InputMethod:      tracker.InputMethod,
			TypingDurationMs: uint64(duration),
			KeystrokeCount:   tracker.KeystrokeCount,
			Metadata:         createEventMetadata(),
		}

		// Set the UI element in metadata
		if tracker.Element != nil {
			event.Metadata.UIElement = tracker.Element
		}

		// Call the callback to emit the event
		if tim.EventCallback != nil {
			go tim.EventCallback(event)
		}

		log.Printf("Text input completed: '%s' (%d keystrokes, %dms, %s, reason: %s)",
			finalText, tracker.KeystrokeCount, duration, tracker.InputMethod, reason)
	}
}

// Helper methods
func (tim *TextInputManager) getElementKey(element *UIElement) string {
	if element == nil {
		return ""
	}
	return element.Role + "|" + element.Name + "|" + element.WindowTitle
}

func (tim *TextInputManager) getCurrentText(element *UIElement) string {
	if element == nil {
		return ""
	}
	// This would need to be implemented using Windows accessibility APIs
	// For now, return empty string as a placeholder
	return ""
}

func (tim *TextInputManager) getFieldName(element *UIElement) string {
	if element == nil || element.Name == "" {
		return "Unknown"
	}
	return element.Name
}

func (tim *TextInputManager) getFieldType(element *UIElement) string {
	if element == nil {
		return "Unknown"
	}
	return element.Role
}

func (tim *TextInputManager) isElementFocused(element *UIElement) bool {
	// This would need to be implemented using Windows accessibility APIs
	// For now, return true as a placeholder
	return true
}

func (tim *TextInputManager) isLikelyPasted(keyCode uint32, char string, tracker *TextInputTracker) bool {
	// Detect Ctrl+V or large text input in short time
	if keyCode == 0x56 { // V key
		return true
	}

	// If many characters appeared in very short time, likely pasted
	timeSinceLastKeystroke := time.Since(tracker.LastKeystroke)
	if timeSinceLastKeystroke < 50*time.Millisecond && len(char) > 1 {
		return true
	}

	return false
}

func (tim *TextInputManager) isLikelySuggestion(keyCode uint32, char string, tracker *TextInputTracker) bool {
	// Detect Tab or Enter after typing started (suggesting autocomplete acceptance)
	if keyCode == 0x09 || keyCode == 0x0D { // Tab or Enter
		return tracker.KeystrokeCount > 0
	}

	// Detect arrow keys followed by Enter (dropdown selection)
	if keyCode == 0x0D && tracker.KeystrokeCount > 2 { // Enter after some typing
		return true
	}

	return false
}

// IsTextInputElement checks if an element is a text input field
func IsTextInputElement(element *UIElement) bool {
	if element == nil {
		return false
	}

	role := strings.ToLower(element.Role)
	name := strings.ToLower(element.Name)

	// Check for common text input roles
	textInputRoles := []string{
		"edit", "text", "textbox", "textarea", "input",
		"searchbox", "passwordbox", "combobox", "document",
	}

	for _, inputRole := range textInputRoles {
		if strings.Contains(role, inputRole) {
			return true
		}
	}

	// Check for common text input names/labels
	textInputNames := []string{
		"text", "input", "search", "email", "password",
		"username", "message", "comment", "description",
	}

	for _, inputName := range textInputNames {
		if strings.Contains(name, inputName) {
			return true
		}
	}

	return false
}
