package main

import (
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

// DragDropEvent represents a drag and drop operation
type DragDropEvent struct {
	StartPosition Position      `json:"start_position"`
	EndPosition   Position      `json:"end_position"`
	SourceElement *UIElement    `json:"source_element,omitempty"`
	DataType      string        `json:"data_type,omitempty"`
	Content       string        `json:"content,omitempty"`
	Success       bool          `json:"success"`
	Metadata      EventMetadata `json:"metadata"`
}

// DragDropTracker tracks drag and drop operations
type DragDropTracker struct {
	IsDragging       bool
	DragStartPos     Position
	DragStartTime    time.Time
	DragStartElement *UIElement
	LastDragPos      Position
	MinDragDistance  float64
	EventCallback    func(DragDropEvent)
	Mutex            sync.RWMutex
}

// NewDragDropTracker creates a new drag drop tracker
func NewDragDropTracker(callback func(DragDropEvent)) *DragDropTracker {
	return &DragDropTracker{
		MinDragDistance: 10.0, // Minimum distance to consider it a drag
		EventCallback:   callback,
	}
}

// HandleMouseDown processes mouse down events for potential drag start
func (ddt *DragDropTracker) HandleMouseDown(position Position, button MouseButton, element *UIElement) {
	if button != MouseButtonLeft {
		return
	}

	ddt.Mutex.Lock()
	defer ddt.Mutex.Unlock()

	// Start potential drag operation
	ddt.DragStartPos = position
	ddt.DragStartTime = time.Now()
	ddt.DragStartElement = element
	ddt.LastDragPos = position
	ddt.IsDragging = false // Not yet confirmed as drag
}

// HandleMouseMove processes mouse move events during drag operation
func (ddt *DragDropTracker) HandleMouseMove(position Position) {
	ddt.Mutex.Lock()
	defer ddt.Mutex.Unlock()

	// Check if we have a potential drag start
	if ddt.DragStartTime.IsZero() {
		return
	}

	// Update last drag position
	ddt.LastDragPos = position

	// Check if we've moved enough to constitute a drag
	distance := ddt.calculateDistance(ddt.DragStartPos, position)
	if !ddt.IsDragging && distance >= ddt.MinDragDistance {
		ddt.IsDragging = true
		log.Printf("Drag operation started at (%d, %d)", ddt.DragStartPos.X, ddt.DragStartPos.Y)
	}
}

// HandleMouseUp processes mouse up events to complete drag operation
func (ddt *DragDropTracker) HandleMouseUp(position Position, button MouseButton, element *UIElement) {
	if button != MouseButtonLeft {
		return
	}

	ddt.Mutex.Lock()
	defer ddt.Mutex.Unlock()

	// Check if this was actually a drag operation
	if !ddt.IsDragging || ddt.DragStartTime.IsZero() {
		ddt.clearDragState()
		return
	}

	// Calculate drag distance
	distance := ddt.calculateDistance(ddt.DragStartPos, position)
	if distance < ddt.MinDragDistance {
		ddt.clearDragState()
		return // Not enough movement for a drag
	}

	// Determine if the drop was successful
	dropSuccess := ddt.isSuccessfulDrop(position, element)

	// Try to get drag content and data type
	content, dataType := ddt.getDragContent()

	// Create drag drop event
	event := DragDropEvent{
		StartPosition: ddt.DragStartPos,
		EndPosition:   position,
		SourceElement: ddt.DragStartElement,
		DataType:      dataType,
		Content:       content,
		Success:       dropSuccess,
		Metadata:      createEventMetadata(),
	}

	// Set target element in metadata
	if element != nil {
		event.Metadata.UIElement = element
	}

	// Emit the event
	if ddt.EventCallback != nil {
		go ddt.EventCallback(event)
	}

	duration := time.Since(ddt.DragStartTime)
	log.Printf("Drag and drop completed: (%d,%d) -> (%d,%d) in %v (success: %t)",
		ddt.DragStartPos.X, ddt.DragStartPos.Y,
		position.X, position.Y,
		duration, dropSuccess)

	ddt.clearDragState()
}

// HandleKeyPress processes key events that might cancel drag operations
func (ddt *DragDropTracker) HandleKeyPress(keyCode uint32, isKeyDown bool) {
	// ESC key cancels drag operations
	if keyCode == 0x1B && isKeyDown { // VK_ESCAPE
		ddt.Mutex.Lock()
		defer ddt.Mutex.Unlock()

		if ddt.IsDragging {
			log.Println("Drag operation cancelled by ESC key")
			ddt.clearDragState()
		}
	}
}

// Helper methods
func (ddt *DragDropTracker) calculateDistance(p1, p2 Position) float64 {
	dx := float64(p2.X - p1.X)
	dy := float64(p2.Y - p1.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (ddt *DragDropTracker) clearDragState() {
	ddt.IsDragging = false
	ddt.DragStartTime = time.Time{}
	ddt.DragStartElement = nil
	ddt.DragStartPos = Position{}
	ddt.LastDragPos = Position{}
}

func (ddt *DragDropTracker) isSuccessfulDrop(dropPosition Position, targetElement *UIElement) bool {
	// Heuristics to determine if the drop was successful:

	// 1. Check if dropped on a valid drop target
	if targetElement != nil && ddt.isDropTarget(targetElement) {
		return true
	}

	// 2. Check if drag distance suggests intentional drop
	distance := ddt.calculateDistance(ddt.DragStartPos, dropPosition)
	if distance > 50.0 { // Significant movement suggests intentional drop
		return true
	}

	// 3. Check drag duration (longer drags are more likely to be intentional)
	duration := time.Since(ddt.DragStartTime)
	if duration > 500*time.Millisecond {
		return true
	}

	return false
}

func (ddt *DragDropTracker) isDropTarget(element *UIElement) bool {
	if element == nil {
		return false
	}

	// Common drop target roles
	dropTargetRoles := []string{
		"list", "listbox", "tree", "treeitem",
		"document", "application", "pane", "window",
		"textbox", "editbox", "richedit",
		"button", "dropdownbutton",
	}

	elementRole := strings.ToLower(element.Role)
	for _, role := range dropTargetRoles {
		if elementRole == role || strings.Contains(elementRole, role) {
			return true
		}
	}

	// Check for drop-related keywords in the element name
	elementName := strings.ToLower(element.Name)
	dropKeywords := []string{
		"drop", "upload", "import", "attach", "browse",
		"file", "folder", "directory",
	}

	for _, keyword := range dropKeywords {
		if strings.Contains(elementName, keyword) {
			return true
		}
	}

	return false
}

func (ddt *DragDropTracker) getDragContent() (content string, dataType string) {
	// This is a simplified implementation. In a production system,
	// you would need to:
	// 1. Access Windows IDataObject interface during drag operations
	// 2. Query available formats (CF_TEXT, CF_HDROP, etc.)
	// 3. Extract actual content being dragged

	// For file drags, check clipboard for file paths
	clipboardContent := getClipboardContent()
	if clipboardContent != "" {
		// Simple heuristic: if clipboard contains file paths
		if strings.Contains(clipboardContent, ":\\") || strings.Contains(clipboardContent, "/") {
			return clipboardContent, "file"
		}
		return clipboardContent, "text"
	}

	// Check for common drag sources based on start element
	if ddt.DragStartElement != nil {
		elementRole := strings.ToLower(ddt.DragStartElement.Role)
		elementName := strings.ToLower(ddt.DragStartElement.Name)

		// File explorer or desktop
		if strings.Contains(elementRole, "listitem") || strings.Contains(elementRole, "treeitem") {
			if strings.Contains(elementName, ".") { // Likely a filename
				return ddt.DragStartElement.Name, "file"
			}
		}

		// Text elements
		if strings.Contains(elementRole, "text") || strings.Contains(elementRole, "edit") {
			return ddt.DragStartElement.Name, "text"
		}

		// Images
		if strings.Contains(elementRole, "image") || strings.Contains(elementName, "image") {
			return ddt.DragStartElement.Name, "image"
		}
	}

	return "", "unknown"
}

// GetCurrentDragInfo returns information about any active drag operation
func (ddt *DragDropTracker) GetCurrentDragInfo() (isDragging bool, startPos Position, currentPos Position, duration time.Duration) {
	ddt.Mutex.RLock()
	defer ddt.Mutex.RUnlock()

	return ddt.IsDragging, ddt.DragStartPos, ddt.LastDragPos, time.Since(ddt.DragStartTime)
}
