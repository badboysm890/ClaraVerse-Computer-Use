package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// EnhancedWorkflowRecorder combines all the new features into a unified recorder
type EnhancedWorkflowRecorder struct {
	Config               *EnhancedWorkflowRecorderConfig
	TextInputManager     *TextInputManager
	BrowserTabTracker    *BrowserTabTracker
	HotkeyDetector       *HotkeyDetector
	TextSelectionTracker *TextSelectionTracker
	DragDropTracker      *DragDropTracker
	RateLimiter          *RateLimiter

	// Event recording
	Events      []WorkflowEvent
	EventsMutex sync.RWMutex
	StartTime   time.Time
	IsRecording bool

	// Performance monitoring
	LastEventTime      time.Time
	EventCount         int64
	FilteredEventCount int64
}

// NewEnhancedWorkflowRecorder creates a new enhanced workflow recorder
func NewEnhancedWorkflowRecorder(config *EnhancedWorkflowRecorderConfig) (*EnhancedWorkflowRecorder, error) {
	// Validate configuration
	if err := ValidateEnhancedConfig(config); err != nil {
		return nil, err
	}

	recorder := &EnhancedWorkflowRecorder{
		Config:    config,
		Events:    make([]WorkflowEvent, 0),
		StartTime: time.Now(),
	}

	// Initialize all trackers with unified event handling
	recorder.TextInputManager = NewTextInputManager(
		3*time.Second, // 3 second completion timeout
		recorder.handleTextInputEvent,
	)

	recorder.BrowserTabTracker = NewBrowserTabTracker(
		recorder.handleBrowserNavigationEvent,
	)

	recorder.HotkeyDetector = NewHotkeyDetector(
		recorder.handleHotkeyEvent,
	)

	recorder.TextSelectionTracker = NewTextSelectionTracker(
		recorder.handleTextSelectionEvent,
	)

	recorder.DragDropTracker = NewDragDropTracker(
		recorder.handleDragDropEvent,
	)

	// Create rate limiter if configured
	recorder.RateLimiter = config.CreateRateLimiter()

	return recorder, nil
}

// StartRecording begins the enhanced workflow recording
func (ewr *EnhancedWorkflowRecorder) StartRecording() error {
	if ewr.IsRecording {
		return NewWorkflowError(ErrorTypeRecording, "Recording already in progress", nil)
	}

	ewr.IsRecording = true
	ewr.StartTime = time.Now()
	ewr.Events = make([]WorkflowEvent, 0)

	log.Printf("Enhanced workflow recording started with %s performance mode", ewr.Config.PerformanceMode)
	ewr.Config.LogPerformanceSettings()

	return nil
}

// StopRecording stops the workflow recording
func (ewr *EnhancedWorkflowRecorder) StopRecording() {
	if !ewr.IsRecording {
		return
	}

	ewr.IsRecording = false

	// Complete any active text input sessions
	ewr.TextInputManager.CompleteAllActiveInputs()

	duration := time.Since(ewr.StartTime)
	log.Printf("Enhanced workflow recording stopped after %s", FormatDuration(duration))
	log.Printf("Recorded %d events (%d filtered out)", ewr.EventCount, ewr.FilteredEventCount)
}

// Event handlers for different event types

func (ewr *EnhancedWorkflowRecorder) handleTextInputEvent(event TextInputCompletedEvent) {
	if !ewr.IsRecording {
		return
	}

	if ewr.shouldRecordEvent(event) {
		ewr.addEvent(event)
		log.Printf("Text input completed: '%s' via %s",
			TruncateString(event.TextValue, 50, "..."), event.InputMethod)
	}
}

func (ewr *EnhancedWorkflowRecorder) handleBrowserNavigationEvent(event BrowserTabNavigationEvent) {
	if !ewr.IsRecording {
		return
	}

	if ewr.shouldRecordEvent(event) {
		ewr.addEvent(event)
		log.Printf("Browser navigation: %s %s -> %s",
			event.Browser,
			TruncateString(event.FromURL, 50, "..."),
			TruncateString(event.ToURL, 50, "..."))
	}
}

func (ewr *EnhancedWorkflowRecorder) handleHotkeyEvent(event HotkeyEvent) {
	if !ewr.IsRecording {
		return
	}

	// Process hotkey for other trackers
	currentElement := getCurrentUIElement()
	ewr.BrowserTabTracker.HandleHotkey(event.Combination, currentElement)
	ewr.TextSelectionTracker.HandleKeyboardShortcut(event.Combination)

	if ewr.shouldRecordEvent(event) {
		ewr.addEvent(event)
		log.Printf("Hotkey detected: %s (%s)", event.Combination, event.Action)
	}
}

func (ewr *EnhancedWorkflowRecorder) handleTextSelectionEvent(event TextSelectionEvent) {
	if !ewr.IsRecording {
		return
	}

	if ewr.shouldRecordEvent(event) {
		ewr.addEvent(event)
		log.Printf("Text selection: '%s' via %s",
			TruncateString(event.SelectedText, 50, "..."), event.SelectionMethod)
	}
}

func (ewr *EnhancedWorkflowRecorder) handleDragDropEvent(event DragDropEvent) {
	if !ewr.IsRecording {
		return
	}

	if ewr.shouldRecordEvent(event) {
		ewr.addEvent(event)
		log.Printf("Drag & drop: %s content from (%d,%d) to (%d,%d), success: %t",
			event.DataType,
			event.StartPosition.X, event.StartPosition.Y,
			event.EndPosition.X, event.EndPosition.Y,
			event.Success)
	}
}

// Enhanced mouse event handling that integrates with all trackers
func (ewr *EnhancedWorkflowRecorder) HandleMouseEvent(eventType MouseEventType, button MouseButton, position Position, scrollDelta *[2]int32) {
	if !ewr.IsRecording {
		return
	}

	currentElement := getCurrentUIElement()

	// Pass to trackers
	switch eventType {
	case MouseDown:
		ewr.TextSelectionTracker.HandleMouseDown(position, button)
		ewr.DragDropTracker.HandleMouseDown(position, button, currentElement)
	case MouseMove:
		ewr.TextSelectionTracker.HandleMouseMove(position)
		ewr.DragDropTracker.HandleMouseMove(position)
	case MouseUp:
		ewr.TextSelectionTracker.HandleMouseUp(position, button)
		ewr.DragDropTracker.HandleMouseUp(position, button, currentElement)
	case MouseClick:
		ewr.BrowserTabTracker.HandleClick(position, currentElement)
	}

	// Create mouse event
	mouseEvent := MouseEvent{
		EventType:   eventType,
		Button:      button,
		Position:    position,
		ScrollDelta: scrollDelta,
		Metadata:    createEventMetadata(),
	}

	if ewr.shouldRecordEvent(mouseEvent) {
		ewr.addEvent(mouseEvent)
	}
}

// Enhanced keyboard event handling
func (ewr *EnhancedWorkflowRecorder) HandleKeyboardEvent(keyCode uint32, isKeyDown bool, character *string) {
	if !ewr.IsRecording {
		return
	}

	// Pass to trackers
	ewr.HotkeyDetector.HandleKeyPress(keyCode, isKeyDown)
	ewr.DragDropTracker.HandleKeyPress(keyCode, isKeyDown)

	if character != nil && *character != "" {
		ewr.TextInputManager.HandleKeystroke(keyCode, *character)
	}

	// Create keyboard event
	keyboardEvent := KeyboardEvent{
		KeyCode:        keyCode,
		IsKeyDown:      isKeyDown,
		ModifierStates: getCurrentModifierStates(),
		Character:      character,
		Metadata:       createEventMetadata(),
	}

	if ewr.shouldRecordEvent(keyboardEvent) {
		ewr.addEvent(keyboardEvent)
	}
}

// Window change handling for application switches and browser navigation
func (ewr *EnhancedWorkflowRecorder) HandleWindowChange() {
	if !ewr.IsRecording {
		return
	}

	currentElement := getCurrentUIElement()
	if currentElement == nil {
		return
	}

	// Pass to trackers
	ewr.BrowserTabTracker.HandleWindowChange(currentElement)

	// Check for text input elements
	if IsTextInputElement(currentElement) {
		ewr.TextInputManager.StartTextInput(currentElement)
	}
}

// Utility methods

func (ewr *EnhancedWorkflowRecorder) shouldRecordEvent(event interface{}) bool {
	// Apply rate limiting if configured
	if ewr.RateLimiter != nil && !ewr.RateLimiter.Allow() {
		ewr.FilteredEventCount++
		return false
	}

	// Apply performance-based filtering
	if ewr.Config.ShouldFilterEvent(event) {
		ewr.FilteredEventCount++
		return false
	}

	return true
}

func (ewr *EnhancedWorkflowRecorder) addEvent(event interface{}) {
	ewr.EventsMutex.Lock()
	defer ewr.EventsMutex.Unlock()

	ewr.Events = append(ewr.Events, event)
	ewr.EventCount++
	ewr.LastEventTime = time.Now()
}

// SaveWorkflow saves the recorded workflow to a JSON file
func (ewr *EnhancedWorkflowRecorder) SaveWorkflow(name string) error {
	if len(ewr.Events) == 0 {
		return NewWorkflowError(ErrorTypeSerialization, "No events to save", nil)
	}

	workflow := RecordedWorkflow{
		Name:      name,
		StartTime: uint64(ewr.StartTime.UnixNano() / int64(time.Millisecond)),
		EndTime:   GetCurrentTimestamp(),
		Events:    ewr.Events,
	}

	filename := GenerateWorkflowFilename(name, "json")
	return SaveJSONToFile(workflow, filename)
}

// GetStatistics returns recording statistics
func (ewr *EnhancedWorkflowRecorder) GetStatistics() map[string]interface{} {
	ewr.EventsMutex.RLock()
	defer ewr.EventsMutex.RUnlock()

	stats := map[string]interface{}{
		"recording_duration": FormatDuration(time.Since(ewr.StartTime)),
		"total_events":       ewr.EventCount,
		"filtered_events":    ewr.FilteredEventCount,
		"events_in_memory":   len(ewr.Events),
		"performance_mode":   ewr.Config.PerformanceMode.String(),
		"is_recording":       ewr.IsRecording,
	}

	// Event type breakdown
	eventTypes := make(map[string]int)
	for _, event := range ewr.Events {
		switch event.(type) {
		case MouseEvent:
			eventTypes["mouse"]++
		case KeyboardEvent:
			eventTypes["keyboard"]++
		case TextInputCompletedEvent:
			eventTypes["text_input"]++
		case BrowserTabNavigationEvent:
			eventTypes["browser_navigation"]++
		case HotkeyEvent:
			eventTypes["hotkey"]++
		case TextSelectionEvent:
			eventTypes["text_selection"]++
		case DragDropEvent:
			eventTypes["drag_drop"]++
		default:
			eventTypes["other"]++
		}
	}
	stats["event_types"] = eventTypes

	return stats
}

// Helper function to get string value from pointer safely
func SafeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Example usage function
func RunEnhancedWorkflowRecorderExample() {
	// Create enhanced configuration
	config := NewBalancedConfig()     // or NewLowEnergyConfig() for better performance
	config.CaptureScreenshots = false // Disable for this example

	// Create recorder
	recorder, err := NewEnhancedWorkflowRecorder(&config)
	if err != nil {
		log.Fatalf("Failed to create enhanced recorder: %v", err)
	}

	// Start recording
	if err := recorder.StartRecording(); err != nil {
		log.Fatalf("Failed to start recording: %v", err)
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Simulate some events (in real usage, these would come from the system)
	go func() {
		// Simulate mouse events
		recorder.HandleMouseEvent(MouseClick, MouseButtonLeft, Position{X: 100, Y: 200}, nil)
		time.Sleep(100 * time.Millisecond)

		// Simulate keyboard events
		recorder.HandleKeyboardEvent(0x41, true, func() *string { s := "a"; return &s }()) // 'a' key down
		time.Sleep(50 * time.Millisecond)
		recorder.HandleKeyboardEvent(0x41, false, nil) // 'a' key up

		// Simulate window change
		recorder.HandleWindowChange()
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received")

	// Stop recording
	recorder.StopRecording()

	// Print statistics
	stats := recorder.GetStatistics()
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Printf("Recording Statistics:\n%s\n", statsJSON)

	// Save workflow
	if err := recorder.SaveWorkflow("example_enhanced_workflow"); err != nil {
		log.Printf("Failed to save workflow: %v", err)
	} else {
		log.Println("Workflow saved successfully")
	}
}

// getCurrentModifierStates returns the current state of modifier keys
func getCurrentModifierStates() ModifierStates {
	return ModifierStates{
		Ctrl:  isKeyPressed(VK_CONTROL),
		Alt:   isKeyPressed(VK_MENU),
		Shift: isKeyPressed(VK_SHIFT),
		Win:   isKeyPressed(VK_LWIN) || isKeyPressed(VK_RWIN),
	}
}
