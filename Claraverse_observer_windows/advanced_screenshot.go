package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"time"
	"unsafe"
)

// Additional screenshot triggers beyond the basic ones already defined
const (
	ScreenshotTriggerManual   ScreenshotTrigger = "Manual"
	ScreenshotTriggerUIChange ScreenshotTrigger = "UIChange"
)

// Monitor information structure
type MonitorInfo struct {
	Name   string
	Width  int32
	Height int32
	Left   int32
	Top    int32
}

// Enhanced screenshot configuration
type AdvancedScreenshotConfig struct {
	EnableAdvancedCompression bool
	PreferredFormat           string // "jpeg", "png"
	JpegQuality               int    // 1-100
	EnableMonitorDetection    bool
	ScaleDownLargeImages      bool
	MaxImageSize              int // bytes
}

// Windows API for monitor enumeration (reuse existing user32)
var (
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
	procMonitorFromWindow   = user32.NewProc("MonitorFromWindow")
)

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type MONITORINFO struct {
	cbSize    uint32
	rcMonitor RECT
	rcWork    RECT
	dwFlags   uint32
}

// Enhanced screenshot capture with advanced features
func captureAdvancedScreenshot(trigger ScreenshotTrigger) *ScreenshotEvent {
	config := globalState.Config

	// Check if screenshots are enabled
	if !config.CaptureScreenshots {
		return nil
	}

	// Rate limiting for screenshots
	now := time.Now()
	if now.Sub(globalState.LastScreenshotTime) < time.Duration(100)*time.Millisecond {
		return nil
	}
	globalState.LastScreenshotTime = now

	// Get current monitor info
	monitor := getCurrentMonitorInfo()

	// Capture the screenshot using existing method
	screenshot := captureScreenshot(trigger)
	if screenshot == nil {
		return nil
	}

	// Apply advanced processing
	advancedConfig := AdvancedScreenshotConfig{
		EnableAdvancedCompression: true,
		PreferredFormat:           config.ScreenshotFormat,
		JpegQuality:               config.ScreenshotJPEGQuality,
		EnableMonitorDetection:    true,
		ScaleDownLargeImages:      true,
		MaxImageSize:              500000, // 500KB limit
	}

	// Enhance the screenshot with advanced features
	enhancedScreenshot := enhanceScreenshot(screenshot, monitor, advancedConfig)

	return enhancedScreenshot
}

// Get current monitor information
func getCurrentMonitorInfo() MonitorInfo {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return MonitorInfo{Name: "Primary", Width: 1920, Height: 1080, Left: 0, Top: 0}
	}

	hMonitor, _, _ := procMonitorFromWindow.Call(hwnd, 0x00000002) // MONITOR_DEFAULTTONEAREST
	if hMonitor == 0 {
		return MonitorInfo{Name: "Primary", Width: 1920, Height: 1080, Left: 0, Top: 0}
	}

	var mi MONITORINFO
	mi.cbSize = uint32(unsafe.Sizeof(mi))

	ret, _, _ := procGetMonitorInfo.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if ret == 0 {
		return MonitorInfo{Name: "Primary", Width: 1920, Height: 1080, Left: 0, Top: 0}
	}

	return MonitorInfo{
		Name:   "Monitor",
		Width:  mi.rcMonitor.Right - mi.rcMonitor.Left,
		Height: mi.rcMonitor.Bottom - mi.rcMonitor.Top,
		Left:   mi.rcMonitor.Left,
		Top:    mi.rcMonitor.Top,
	}
}

// Enhance screenshot with advanced processing
func enhanceScreenshot(screenshot *ScreenshotEvent, monitor MonitorInfo, config AdvancedScreenshotConfig) *ScreenshotEvent {
	if screenshot == nil {
		return nil
	}

	// Decode the base64 image
	imageData, err := base64.StdEncoding.DecodeString(screenshot.ImageBase64)
	if err != nil {
		return screenshot // Return original on error
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return screenshot // Return original on error
	}

	// Apply size limits if configured
	if globalState.Config.MaxScreenshotWidth != nil || globalState.Config.MaxScreenshotHeight != nil {
		img = applySizeLimits(img, globalState.Config)
	}

	// Re-encode with preferred format and compression
	var buf bytes.Buffer
	var finalFormat string

	if config.PreferredFormat == "jpeg" || config.PreferredFormat == "jpg" {
		options := &jpeg.Options{Quality: config.JpegQuality}
		err = jpeg.Encode(&buf, img, options)
		finalFormat = "jpeg"
	} else {
		err = png.Encode(&buf, img)
		finalFormat = "png"
	}

	if err != nil {
		return screenshot // Return original on error
	}

	// Check size limits
	if config.ScaleDownLargeImages && buf.Len() > config.MaxImageSize {
		// If still too large, reduce quality or scale down further
		return screenshot // For now, return original
	}

	// Create enhanced screenshot event
	bounds := img.Bounds()
	enhancedScreenshot := &ScreenshotEvent{
		ImageBase64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		ImageFormat: finalFormat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		MonitorName: monitor.Name,
		Trigger:     screenshot.Trigger,
		Metadata:    screenshot.Metadata,
	}

	return enhancedScreenshot
}

// Check if we should capture screenshot based on trigger type
func shouldCaptureScreenshot(trigger ScreenshotTrigger) bool {
	config := globalState.Config

	switch trigger {
	case ScreenshotTriggerMouseClick:
		return config.ScreenshotOnMouseClick
	case ScreenshotTriggerKeyboard:
		return config.ScreenshotOnKeyboardEvent
	case ScreenshotTriggerInterval:
		return config.ScreenshotOnInterval
	case ScreenshotTriggerAppSwitch:
		return config.ScreenshotOnAppSwitch
	case ScreenshotTriggerManual:
		return true
	case ScreenshotTriggerUIChange:
		return true // Always capture on UI changes if requested
	default:
		return false
	}
}

// Enhanced screenshot trigger for UI changes
func triggerUIChangeScreenshot() {
	if shouldCaptureScreenshot(ScreenshotTriggerUIChange) {
		screenshot := captureAdvancedScreenshot(ScreenshotTriggerUIChange)
		if screenshot != nil {
			// Add to event queue (would need to integrate with main event system)
			logScreenshotEvent(screenshot)
		}
	}
}

// Enhanced screenshot trigger for manual capture
func triggerManualScreenshot() *ScreenshotEvent {
	return captureAdvancedScreenshot(ScreenshotTriggerManual)
}

// Log screenshot event (placeholder for integration with main event system)
func logScreenshotEvent(screenshot *ScreenshotEvent) {
	// This would integrate with the main event logging system
	// For now, this is a placeholder
}

// Detect significant UI changes that warrant a screenshot
func detectUIChange() bool {
	// Simple implementation - check if window has changed
	windowTitle, processID := getCurrentWindow()

	changed := windowTitle != globalState.CurrentWindowTitle ||
		processID != globalState.CurrentProcessID

	if changed {
		globalState.CurrentWindowTitle = windowTitle
		globalState.CurrentProcessID = processID
	}

	return changed
}

// Enhanced screenshot interval processing
func processScreenshotInterval() {
	if !globalState.Config.ScreenshotOnInterval {
		return
	}

	now := time.Now()
	interval := time.Duration(globalState.Config.ScreenshotIntervalMs) * time.Millisecond

	if now.Sub(globalState.LastScreenshotTime) >= interval {
		screenshot := captureAdvancedScreenshot(ScreenshotTriggerInterval)
		if screenshot != nil {
			logScreenshotEvent(screenshot)
		}
	}
}
