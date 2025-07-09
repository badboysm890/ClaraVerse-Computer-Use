package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowText              = user32.NewProc("GetWindowTextW")
	procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
)

const (
	CF_TEXT        = 1
	CF_UNICODETEXT = 13
	VK_LBUTTON     = 0x01
	VK_RBUTTON     = 0x02
	VK_MBUTTON     = 0x04
	VK_CONTROL     = 0x11
	VK_MENU        = 0x12
	VK_SHIFT       = 0x10
	VK_LWIN        = 0x5B
	VK_RWIN        = 0x5C
	VK_SPACE       = 0x20
	VK_RETURN      = 0x0D
)

type POINT struct {
	X int32
	Y int32
}

// Enhanced Configuration System
type PerformanceMode int

const (
	Normal PerformanceMode = iota
	Balanced
	LowEnergy
)

type WorkflowRecorderConfig struct {
	RecordMouse                   bool
	RecordKeyboard                bool
	CaptureUIElements             bool
	RecordClipboard               bool
	RecordHotkeys                 bool
	RecordTextInputCompletion     bool
	RecordApplicationSwitches     bool
	RecordBrowserTabNavigation    bool
	AppSwitchDwellTimeThresholdMs int64
	BrowserDetectionTimeoutMs     int64
	MaxClipboardContentLength     int
	MouseMoveThrottleMs           int64
	MinDragDistance               float64
	PerformanceMode               PerformanceMode
	EventProcessingDelayMs        *int64
	MaxEventsPerSecond            *int32
	FilterMouseNoise              bool
	FilterKeyboardNoise           bool
	ReduceUIElementCapture        bool
	CaptureScreenshots            bool
	ScreenshotOnMouseClick        bool
	ScreenshotOnKeyboardEvent     bool
	ScreenshotOnInterval          bool
	ScreenshotIntervalMs          int64
	ScreenshotOnAppSwitch         bool
	ScreenshotFormat              string
	ScreenshotJPEGQuality         int
	MaxScreenshotWidth            *int
	MaxScreenshotHeight           *int
	IgnoreFocusPatterns           []string
	IgnoreWindowTitles            []string
	IgnoreApplications            []string
}

func DefaultConfig() WorkflowRecorderConfig {
	return WorkflowRecorderConfig{
		RecordMouse:                   true,
		RecordKeyboard:                true,
		CaptureUIElements:             true,
		RecordClipboard:               true,
		RecordHotkeys:                 true,
		RecordTextInputCompletion:     true,
		RecordApplicationSwitches:     true,
		RecordBrowserTabNavigation:    true,
		AppSwitchDwellTimeThresholdMs: 100,
		BrowserDetectionTimeoutMs:     1000,
		MaxClipboardContentLength:     10240,
		MouseMoveThrottleMs:           100,
		MinDragDistance:               5.0,
		PerformanceMode:               Normal,
		FilterMouseNoise:              false,
		FilterKeyboardNoise:           false,
		ReduceUIElementCapture:        false,
		CaptureScreenshots:            true,
		ScreenshotOnMouseClick:        true,
		ScreenshotOnKeyboardEvent:     false,
		ScreenshotOnInterval:          false,
		ScreenshotIntervalMs:          5000,
		ScreenshotOnAppSwitch:         true,
		ScreenshotFormat:              "png",
		ScreenshotJPEGQuality:         85,
		IgnoreFocusPatterns: []string{
			"notification", "tooltip", "popup",
			"sharing your screen", "recording screen", "screen capture",
			"1password", "lastpass", "bitwarden",
			"battery", "volume", "network", "wifi",
		},
		IgnoreWindowTitles: []string{
			"Task Manager", "System Tray", "Hidden Icons",
		},
		IgnoreApplications: []string{
			"dwm.exe", "winlogon.exe", "csrss.exe",
		},
	}
}

// Event structures
type Position struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type UIElement struct {
	Role            string     `json:"role"`
	Name            string     `json:"name"`
	Bounds          [4]float64 `json:"bounds"`
	ProcessID       uint32     `json:"process_id"`
	WindowTitle     string     `json:"window_title"`
	ApplicationName string     `json:"application_name"`
	URL             string     `json:"url,omitempty"`
}

type EventMetadata struct {
	UIElement *UIElement `json:"ui_element,omitempty"`
	Timestamp uint64     `json:"timestamp"`
}

type MouseButton string

const (
	MouseButtonLeft   MouseButton = "Left"
	MouseButtonRight  MouseButton = "Right"
	MouseButtonMiddle MouseButton = "Middle"
	MouseButtonNone   MouseButton = "None"
)

type MouseEventType string

const (
	MouseClick       MouseEventType = "Click"
	MouseDoubleClick MouseEventType = "DoubleClick"
	MouseRightClick  MouseEventType = "RightClick"
	MouseDown        MouseEventType = "Down"
	MouseUp          MouseEventType = "Up"
	MouseMove        MouseEventType = "Move"
	MouseWheel       MouseEventType = "Wheel"
	MouseDrag        MouseEventType = "Drag"
	MouseDragStart   MouseEventType = "DragStart"
	MouseDragEnd     MouseEventType = "DragEnd"
	MouseDrop        MouseEventType = "Drop"
)

type MouseEvent struct {
	EventType   MouseEventType `json:"event_type"`
	Button      MouseButton    `json:"button"`
	Position    Position       `json:"position"`
	ScrollDelta *[2]int32      `json:"scroll_delta,omitempty"`
	DragStart   *Position      `json:"drag_start,omitempty"`
	Metadata    EventMetadata  `json:"metadata"`
}

type ModifierStates struct {
	Ctrl  bool `json:"ctrl"`
	Alt   bool `json:"alt"`
	Shift bool `json:"shift"`
	Win   bool `json:"win"`
}

type KeyboardEvent struct {
	KeyCode        uint32         `json:"key_code"`
	IsKeyDown      bool           `json:"is_key_down"`
	ModifierStates ModifierStates `json:"modifier_states"`
	Character      *string        `json:"character,omitempty"`
	Metadata       EventMetadata  `json:"metadata"`
}

type ClipboardAction string

const (
	ClipboardCopy  ClipboardAction = "Copy"
	ClipboardCut   ClipboardAction = "Cut"
	ClipboardPaste ClipboardAction = "Paste"
	ClipboardClear ClipboardAction = "Clear"
)

type ClipboardEvent struct {
	Action      ClipboardAction `json:"action"`
	Content     string          `json:"content"`
	ContentSize int             `json:"content_size"`
	Format      string          `json:"format"`
	Truncated   bool            `json:"truncated"`
	Metadata    EventMetadata   `json:"metadata"`
}

type HotkeyEvent struct {
	Combination string        `json:"combination"`
	Action      string        `json:"action"`
	IsGlobal    bool          `json:"is_global"`
	Metadata    EventMetadata `json:"metadata"`
}

type ApplicationSwitchMethod string

const (
	AppSwitchAltTab             ApplicationSwitchMethod = "AltTab"
	AppSwitchTaskbarClick       ApplicationSwitchMethod = "TaskbarClick"
	AppSwitchWindowsKeyShortcut ApplicationSwitchMethod = "WindowsKeyShortcut"
	AppSwitchStartMenu          ApplicationSwitchMethod = "StartMenu"
	AppSwitchWindowClick        ApplicationSwitchMethod = "WindowClick"
	AppSwitchOther              ApplicationSwitchMethod = "Other"
)

type ApplicationSwitchEvent struct {
	FromApplication string                  `json:"from_application"`
	ToApplication   string                  `json:"to_application"`
	FromProcessID   uint32                  `json:"from_process_id"`
	ToProcessID     uint32                  `json:"to_process_id"`
	SwitchMethod    ApplicationSwitchMethod `json:"switch_method"`
	DwellTimeMs     uint64                  `json:"dwell_time_ms"`
	SwitchCount     uint32                  `json:"switch_count"`
	Metadata        EventMetadata           `json:"metadata"`
}

type ButtonInteractionType string

const (
	ButtonClick          ButtonInteractionType = "Click"
	ButtonSubmit         ButtonInteractionType = "Submit"
	ButtonCancel         ButtonInteractionType = "Cancel"
	ButtonToggle         ButtonInteractionType = "Toggle"
	ButtonDropdownToggle ButtonInteractionType = "DropdownToggle"
)

type ButtonClickEvent struct {
	ButtonText      string                `json:"button_text"`
	InteractionType ButtonInteractionType `json:"interaction_type"`
	ButtonRole      string                `json:"button_role"`
	WasEnabled      bool                  `json:"was_enabled"`
	Position        Position              `json:"position"`
	Metadata        EventMetadata         `json:"metadata"`
}

type ScreenshotTrigger string

const (
	ScreenshotTriggerMouseClick ScreenshotTrigger = "MouseClick"
	ScreenshotTriggerKeyboard   ScreenshotTrigger = "Keyboard"
	ScreenshotTriggerInterval   ScreenshotTrigger = "Interval"
	ScreenshotTriggerAppSwitch  ScreenshotTrigger = "AppSwitch"
)

type ScreenshotEvent struct {
	ImageBase64 string            `json:"image_base64"`
	ImageFormat string            `json:"image_format"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	MonitorName string            `json:"monitor_name"`
	Trigger     ScreenshotTrigger `json:"trigger"`
	Metadata    EventMetadata     `json:"metadata"`
}

type WorkflowEvent interface{}

type RecordedWorkflow struct {
	Name      string          `json:"name"`
	StartTime uint64          `json:"start_time"`
	EndTime   uint64          `json:"end_time"`
	Events    []WorkflowEvent `json:"events"`
}

// Enhanced Global State
type WorkflowState struct {
	Config               WorkflowRecorderConfig
	LastMousePos         Position
	LastMouseMoveTime    time.Time
	LastClipboardContent string
	CurrentApplication   string
	CurrentProcessID     uint32
	CurrentWindowTitle   string
	ActiveKeys           map[uint32]bool
	ModifierStates       ModifierStates
	LastHotkeyTime       time.Time
	IsDragging           bool
	DragStartPos         Position
	DragStartTime        time.Time
	LastScreenshotTime   time.Time
	EventCount           int32
	EventCountResetTime  time.Time
	LastEventTime        time.Time
	Mutex                sync.RWMutex
}

var globalState = &WorkflowState{
	Config:              DefaultConfig(),
	ActiveKeys:          make(map[uint32]bool),
	ModifierStates:      ModifierStates{},
	LastMouseMoveTime:   time.Now(),
	LastHotkeyTime:      time.Now(),
	LastScreenshotTime:  time.Now(),
	EventCountResetTime: time.Now(),
	LastEventTime:       time.Now(),
}

// Helper functions
func captureTimestamp() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

func createEventMetadata() EventMetadata {
	return EventMetadata{
		UIElement: getCurrentUIElement(),
		Timestamp: captureTimestamp(),
	}
}

func getCurrentUIElement() *UIElement {
	pos := getMousePosition()
	windowTitle, processID := getCurrentWindow()

	return &UIElement{
		Role:            "window",
		Name:            windowTitle,
		Bounds:          [4]float64{float64(pos.X), float64(pos.Y), 100, 100},
		ProcessID:       processID,
		WindowTitle:     windowTitle,
		ApplicationName: getCurrentApplicationName(),
		URL:             getCurrentURL(),
	}
}

func getMousePosition() Position {
	var point POINT
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ret == 0 {
		return Position{X: 0, Y: 0}
	}
	return Position{X: point.X, Y: point.Y}
}

func getCurrentWindow() (string, uint32) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", 0
	}

	var processID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))

	textBuf := make([]uint16, 256)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&textBuf[0])), 256)

	return syscall.UTF16ToString(textBuf), processID
}

func getCurrentApplicationName() string {
	windowTitle, _ := getCurrentWindow()
	if windowTitle == "" {
		return "Unknown"
	}
	return windowTitle
}

func getCurrentURL() string {
	windowTitle, _ := getCurrentWindow()
	if strings.Contains(strings.ToLower(windowTitle), "http") {
		return extractURLFromTitle(windowTitle)
	}
	return ""
}

func extractURLFromTitle(title string) string {
	patterns := []string{
		`https?://[^\s]+`,
		`[a-zA-Z0-9-]+\.[a-zA-Z]{2,}[^\s]*`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(title); match != "" {
			return match
		}
	}

	return ""
}

func isMouseButtonPressed(button int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(button))
	return (ret & 0x8000) != 0
}

func isKeyPressed(keyCode uint32) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(keyCode))
	return (ret & 0x8000) != 0
}

func getClipboardContent() string {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	ret, _, _ = procIsClipboardFormatAvailable.Call(CF_UNICODETEXT)
	if ret != 0 {
		handle, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
		if handle != 0 {
			ptr, _, _ := procGlobalLock.Call(handle)
			if ptr != 0 {
				defer procGlobalUnlock.Call(handle)
				return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(ptr))[:])
			}
		}
	}

	ret, _, _ = procIsClipboardFormatAvailable.Call(CF_TEXT)
	if ret != 0 {
		handle, _, _ := procGetClipboardData.Call(CF_TEXT)
		if handle != 0 {
			ptr, _, _ := procGlobalLock.Call(handle)
			if ptr != 0 {
				defer procGlobalUnlock.Call(handle)
				return string((*[1 << 20]byte)(unsafe.Pointer(ptr))[:])
			}
		}
	}

	return ""
}

func captureScreenshot(trigger ScreenshotTrigger) *ScreenshotEvent {
	if !globalState.Config.CaptureScreenshots {
		return nil
	}

	switch trigger {
	case ScreenshotTriggerMouseClick:
		if !globalState.Config.ScreenshotOnMouseClick {
			return nil
		}
	case ScreenshotTriggerKeyboard:
		if !globalState.Config.ScreenshotOnKeyboardEvent {
			return nil
		}
	case ScreenshotTriggerInterval:
		if !globalState.Config.ScreenshotOnInterval {
			return nil
		}
		now := time.Now()
		if now.Sub(globalState.LastScreenshotTime).Milliseconds() < globalState.Config.ScreenshotIntervalMs {
			return nil
		}
		globalState.LastScreenshotTime = now
	case ScreenshotTriggerAppSwitch:
		if !globalState.Config.ScreenshotOnAppSwitch {
			return nil
		}
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Printf("Failed to capture screenshot: %v", err)
		return nil
	}

	finalImg := applySizeLimits(img, globalState.Config)

	var base64Data string
	var buf strings.Builder

	switch globalState.Config.ScreenshotFormat {
	case "jpeg", "jpg":
		encoder := base64.NewEncoder(base64.StdEncoding, &buf)
		err = jpeg.Encode(encoder, finalImg, &jpeg.Options{
			Quality: globalState.Config.ScreenshotJPEGQuality,
		})
		encoder.Close()
	default:
		encoder := base64.NewEncoder(base64.StdEncoding, &buf)
		err = png.Encode(encoder, finalImg)
		encoder.Close()
	}

	if err != nil {
		log.Printf("Failed to encode screenshot: %v", err)
		return nil
	}

	base64Data = buf.String()
	bounds = finalImg.Bounds()

	return &ScreenshotEvent{
		ImageBase64: base64Data,
		ImageFormat: globalState.Config.ScreenshotFormat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		MonitorName: "Primary",
		Trigger:     trigger,
		Metadata:    createEventMetadata(),
	}
}

func applySizeLimits(img image.Image, config WorkflowRecorderConfig) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxWidth := config.MaxScreenshotWidth
	maxHeight := config.MaxScreenshotHeight

	needsScaling := false
	if maxWidth != nil && width > *maxWidth {
		needsScaling = true
	}
	if maxHeight != nil && height > *maxHeight {
		needsScaling = true
	}

	if !needsScaling {
		return img
	}

	log.Printf("Screenshot scaling requested but not implemented. Original: %dx%d, Requested max: %v x %v",
		width, height, maxWidth, maxHeight)

	return img
}

func shouldFilterEvent(event WorkflowEvent) bool {
	config := globalState.Config
	now := time.Now()

	if config.MaxEventsPerSecond != nil {
		globalState.Mutex.Lock()
		if now.Sub(globalState.EventCountResetTime).Seconds() >= 1.0 {
			globalState.EventCount = 0
			globalState.EventCountResetTime = now
		}

		if globalState.EventCount >= *config.MaxEventsPerSecond {
			globalState.Mutex.Unlock()
			return true
		}
		globalState.EventCount++
		globalState.Mutex.Unlock()
	}

	if config.EventProcessingDelayMs != nil && *config.EventProcessingDelayMs > 0 {
		globalState.Mutex.Lock()
		if now.Sub(globalState.LastEventTime).Milliseconds() < *config.EventProcessingDelayMs {
			globalState.Mutex.Unlock()
			return true
		}
		globalState.LastEventTime = now
		globalState.Mutex.Unlock()
	}

	return false
}

func calculateDistance(p1, p2 Position) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func determineButtonInteractionType(element UIElement) ButtonInteractionType {
	name := strings.ToLower(element.Name)
	role := strings.ToLower(element.Role)

	if strings.Contains(role, "hyperlink") || strings.Contains(role, "link") {
		return ButtonClick
	}

	if strings.Contains(name, "dropdown") || strings.Contains(name, "▼") ||
		strings.Contains(name, "expand") || strings.Contains(name, "collapse") {
		return ButtonDropdownToggle
	}

	if strings.Contains(name, "submit") || strings.Contains(name, "save") ||
		strings.Contains(name, "ok") || strings.Contains(name, "apply") ||
		strings.Contains(name, "confirm") {
		return ButtonSubmit
	}

	if strings.Contains(name, "cancel") || strings.Contains(name, "close") ||
		strings.Contains(name, "×") || strings.Contains(name, "dismiss") {
		return ButtonCancel
	}

	if strings.Contains(role, "toggle") || strings.Contains(role, "checkbox") ||
		strings.Contains(role, "radiobutton") || strings.Contains(name, "toggle") {
		return ButtonToggle
	}

	return ButtonClick
}

func shouldIgnoreApplication(appName, windowTitle string) bool {
	appLower := strings.ToLower(appName)
	titleLower := strings.ToLower(windowTitle)

	for _, pattern := range globalState.Config.IgnoreFocusPatterns {
		if strings.Contains(appLower, strings.ToLower(pattern)) ||
			strings.Contains(titleLower, strings.ToLower(pattern)) {
			return true
		}
	}

	for _, ignoreApp := range globalState.Config.IgnoreApplications {
		if strings.Contains(appLower, strings.ToLower(ignoreApp)) {
			return true
		}
	}

	for _, ignoreTitle := range globalState.Config.IgnoreWindowTitles {
		if strings.Contains(titleLower, strings.ToLower(ignoreTitle)) {
			return true
		}
	}

	return false
}

func processClipboardEvents(events *[]WorkflowEvent) {
	currentContent := getClipboardContent()
	if currentContent != globalState.LastClipboardContent && currentContent != "" {
		clipboardEvent := ClipboardEvent{
			Action:      ClipboardCopy,
			Content:     currentContent,
			ContentSize: len(currentContent),
			Format:      "text/plain",
			Truncated:   false,
			Metadata:    createEventMetadata(),
		}

		if !shouldFilterEvent(clipboardEvent) {
			*events = append(*events, clipboardEvent)
			fmt.Printf("📋 Clipboard: %s\n", currentContent[:min(50, len(currentContent))])
		}

		globalState.LastClipboardContent = currentContent
	}
}

func processApplicationSwitchEvents(events *[]WorkflowEvent, element UIElement) {
	currentApp := element.ApplicationName
	if currentApp != globalState.CurrentApplication && currentApp != "" {
		switchEvent := ApplicationSwitchEvent{
			FromApplication: globalState.CurrentApplication,
			ToApplication:   currentApp,
			FromProcessID:   globalState.CurrentProcessID,
			ToProcessID:     element.ProcessID,
			SwitchMethod:    AppSwitchOther,
			DwellTimeMs:     uint64(time.Now().UnixNano()/int64(time.Millisecond)) - captureTimestamp(),
			SwitchCount:     1,
			Metadata:        createEventMetadata(),
		}

		if !shouldFilterEvent(switchEvent) {
			*events = append(*events, switchEvent)

			if screenshot := captureScreenshot(ScreenshotTriggerAppSwitch); screenshot != nil {
				*events = append(*events, *screenshot)
			}

			fmt.Printf("🔄 App Switch: %s -> %s\n", globalState.CurrentApplication, currentApp)
		}

		globalState.CurrentApplication = currentApp
		globalState.CurrentProcessID = element.ProcessID
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func processEnhancedEvents(workflow *RecordedWorkflow) {
	mousePos := getMousePosition()
	windowTitle, processID := getCurrentWindow()
	appName := getCurrentApplicationName()

	element := UIElement{
		Role:            "window",
		Name:            windowTitle,
		Bounds:          [4]float64{float64(mousePos.X), float64(mousePos.Y), 100, 100},
		ProcessID:       processID,
		WindowTitle:     windowTitle,
		ApplicationName: appName,
		URL:             getCurrentURL(),
	}

	if shouldIgnoreApplication(appName, windowTitle) {
		return
	}

	var events []WorkflowEvent

	// Enhanced mouse event processing
	if mousePos.X != globalState.LastMousePos.X || mousePos.Y != globalState.LastMousePos.Y {
		now := time.Now()
		if now.Sub(globalState.LastMouseMoveTime).Milliseconds() >= globalState.Config.MouseMoveThrottleMs {
			mouseEvent := MouseEvent{
				EventType: MouseMove,
				Position:  mousePos,
				Button:    MouseButtonNone,
				Metadata:  createEventMetadata(),
			}

			if !shouldFilterEvent(mouseEvent) {
				events = append(events, mouseEvent)

				if len(workflow.Events)%50 == 0 {
					fmt.Printf("🖱️  Mouse: (%d, %d) in %s\n", mousePos.X, mousePos.Y, windowTitle)
				}
			}

			globalState.LastMousePos = mousePos
			globalState.LastMouseMoveTime = now
		}
	}

	// Enhanced mouse click detection with screenshots
	if isMouseButtonPressed(VK_LBUTTON) {
		if !globalState.IsDragging {
			globalState.IsDragging = true
			globalState.DragStartPos = mousePos
			globalState.DragStartTime = time.Now()
		}
	} else if globalState.IsDragging {
		globalState.IsDragging = false

		dragDistance := calculateDistance(globalState.DragStartPos, mousePos)

		var eventType MouseEventType
		if dragDistance >= globalState.Config.MinDragDistance {
			eventType = MouseDrag
		} else {
			eventType = MouseClick
		}

		mouseEvent := MouseEvent{
			EventType: eventType,
			Position:  mousePos,
			Button:    MouseButtonLeft,
			Metadata:  createEventMetadata(),
		}

		if !shouldFilterEvent(mouseEvent) {
			events = append(events, mouseEvent)

			if screenshot := captureScreenshot(ScreenshotTriggerMouseClick); screenshot != nil {
				events = append(events, *screenshot)
			}

			interactionType := determineButtonInteractionType(element)
			buttonEvent := ButtonClickEvent{
				ButtonText:      element.Name,
				InteractionType: interactionType,
				ButtonRole:      element.Role,
				WasEnabled:      true,
				Position:        mousePos,
				Metadata:        createEventMetadata(),
			}

			if !shouldFilterEvent(buttonEvent) {
				events = append(events, buttonEvent)
			}

			fmt.Printf("🖱️  %s at (%d, %d) - %s (%s)\n",
				eventType, mousePos.X, mousePos.Y, element.Name, interactionType)
		}
	}

	processClipboardEvents(&events)
	processApplicationSwitchEvents(&events, element)

	if screenshot := captureScreenshot(ScreenshotTriggerInterval); screenshot != nil {
		events = append(events, *screenshot)
		fmt.Printf("📸 Interval screenshot captured\n")
	}

	for _, event := range events {
		workflow.Events = append(workflow.Events, event)
	}
}

func main() {
	workflow := &RecordedWorkflow{
		Name:      "Enhanced Workflow Recording",
		StartTime: captureTimestamp(),
		Events:    []WorkflowEvent{},
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🚀 Enhanced UI Workflow Recorder Started")
	fmt.Println("📊 Features: Screenshots, Rate Limiting, Browser Navigation, Performance Modes")
	fmt.Printf("⚙️  Performance Mode: %v\n", globalState.Config.PerformanceMode)
	fmt.Printf("📸 Screenshots: %v (Format: %s)\n", globalState.Config.CaptureScreenshots, globalState.Config.ScreenshotFormat)
	fmt.Println("Press Ctrl+C to stop recording...")

	go func() {
		for {
			select {
			case <-c:
				return
			default:
				processEnhancedEvents(workflow)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	<-c
	fmt.Println("\n🛑 Stopping recorder...")

	workflow.EndTime = captureTimestamp()

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("ui_recording_enhanced_%s.json", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(workflow); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Enhanced recording saved to %s\n", filename)
	fmt.Printf("📊 Total events recorded: %d\n", len(workflow.Events))
	fmt.Printf("⏱️  Recording duration: %.2f seconds\n",
		float64(workflow.EndTime-workflow.StartTime)/1000.0)
}
