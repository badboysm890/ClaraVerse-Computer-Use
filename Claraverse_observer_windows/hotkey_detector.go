package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// HotkeyPattern represents a known hotkey combination
type HotkeyPattern struct {
	Keys        []uint32
	Combination string
	Action      string
	IsGlobal    bool
	Category    string
}

// HotkeyDetector tracks pressed keys and detects hotkey combinations
type HotkeyDetector struct {
	PressedKeys    map[uint32]bool
	KeyPressOrder  []uint32
	LastKeyTime    time.Time
	HotkeyPatterns []HotkeyPattern
	EventCallback  func(HotkeyEvent)
	MaxKeyDelay    time.Duration
	Mutex          sync.RWMutex
}

// NewHotkeyDetector creates a new hotkey detector
func NewHotkeyDetector(callback func(HotkeyEvent)) *HotkeyDetector {
	detector := &HotkeyDetector{
		PressedKeys:    make(map[uint32]bool),
		KeyPressOrder:  make([]uint32, 0),
		HotkeyPatterns: initializeHotkeyPatterns(),
		EventCallback:  callback,
		MaxKeyDelay:    time.Millisecond * 500, // Max delay between keys in a combination
	}

	return detector
}

// HandleKeyPress processes a key press event
func (hd *HotkeyDetector) HandleKeyPress(keyCode uint32, isKeyDown bool) {
	hd.Mutex.Lock()
	defer hd.Mutex.Unlock()

	now := time.Now()

	if isKeyDown {
		// Key pressed down
		if !hd.PressedKeys[keyCode] {
			hd.PressedKeys[keyCode] = true
			hd.KeyPressOrder = append(hd.KeyPressOrder, keyCode)
			hd.LastKeyTime = now

			// Check for hotkey combinations
			hd.checkForHotkeys()
		}
	} else {
		// Key released
		if hd.PressedKeys[keyCode] {
			delete(hd.PressedKeys, keyCode)

			// Remove from press order
			for i, k := range hd.KeyPressOrder {
				if k == keyCode {
					hd.KeyPressOrder = append(hd.KeyPressOrder[:i], hd.KeyPressOrder[i+1:]...)
					break
				}
			}

			// Clear all keys if too much time has passed or no keys are pressed
			if len(hd.PressedKeys) == 0 || time.Since(hd.LastKeyTime) > hd.MaxKeyDelay {
				hd.clearState()
			}
		}
	}
}

// checkForHotkeys checks if the currently pressed keys match any known patterns
func (hd *HotkeyDetector) checkForHotkeys() {
	if len(hd.PressedKeys) < 2 {
		return // Hotkeys need at least 2 keys
	}

	// Create sorted list of pressed keys for comparison
	pressedKeysList := make([]uint32, 0, len(hd.PressedKeys))
	for key := range hd.PressedKeys {
		pressedKeysList = append(pressedKeysList, key)
	}
	sort.Slice(pressedKeysList, func(i, j int) bool {
		return pressedKeysList[i] < pressedKeysList[j]
	})

	// Check against known patterns
	for _, pattern := range hd.HotkeyPatterns {
		if hd.keysMatch(pressedKeysList, pattern.Keys) {
			// Found a matching pattern
			event := HotkeyEvent{
				Combination: pattern.Combination,
				Action:      pattern.Action,
				IsGlobal:    pattern.IsGlobal,
				Metadata:    createEventMetadata(),
			}

			// Emit the event
			if hd.EventCallback != nil {
				go hd.EventCallback(event)
			}

			// Clear state to prevent duplicate events
			hd.clearState()
			return
		}
	}
}

// keysMatch checks if the pressed keys match a pattern
func (hd *HotkeyDetector) keysMatch(pressedKeys, patternKeys []uint32) bool {
	if len(pressedKeys) != len(patternKeys) {
		return false
	}

	// Sort pattern keys for comparison
	sortedPatternKeys := make([]uint32, len(patternKeys))
	copy(sortedPatternKeys, patternKeys)
	sort.Slice(sortedPatternKeys, func(i, j int) bool {
		return sortedPatternKeys[i] < sortedPatternKeys[j]
	})

	// Compare sorted arrays
	for i, key := range pressedKeys {
		if key != sortedPatternKeys[i] {
			return false
		}
	}

	return true
}

// clearState clears the current key tracking state
func (hd *HotkeyDetector) clearState() {
	hd.PressedKeys = make(map[uint32]bool)
	hd.KeyPressOrder = make([]uint32, 0)
}

// GetCurrentCombination returns the current key combination as a string
func (hd *HotkeyDetector) GetCurrentCombination() string {
	hd.Mutex.RLock()
	defer hd.Mutex.RUnlock()

	if len(hd.PressedKeys) == 0 {
		return ""
	}

	var keys []string
	var modifiers []string
	var regularKeys []string

	for keyCode := range hd.PressedKeys {
		keyName := hd.getKeyName(keyCode)
		if hd.isModifierKey(keyCode) {
			modifiers = append(modifiers, keyName)
		} else {
			regularKeys = append(regularKeys, keyName)
		}
	}

	// Sort for consistent output
	sort.Strings(modifiers)
	sort.Strings(regularKeys)

	// Combine modifiers first, then regular keys
	keys = append(keys, modifiers...)
	keys = append(keys, regularKeys...)

	return strings.Join(keys, "+")
}

// Helper methods
func (hd *HotkeyDetector) isModifierKey(keyCode uint32) bool {
	modifierKeys := []uint32{
		VK_CONTROL, VK_MENU, VK_SHIFT, VK_LWIN, VK_RWIN,
		0xA2, 0xA3, // VK_LCONTROL, VK_RCONTROL
		0xA4, 0xA5, // VK_LMENU, VK_RMENU
		0xA0, 0xA1, // VK_LSHIFT, VK_RSHIFT
	}

	for _, modKey := range modifierKeys {
		if keyCode == modKey {
			return true
		}
	}

	return false
}

func (hd *HotkeyDetector) getKeyName(keyCode uint32) string {
	keyNames := map[uint32]string{
		// Modifier keys
		VK_CONTROL: "Ctrl",
		VK_MENU:    "Alt",
		VK_SHIFT:   "Shift",
		VK_LWIN:    "Win",
		VK_RWIN:    "Win",
		0xA2:       "Ctrl",  // VK_LCONTROL
		0xA3:       "Ctrl",  // VK_RCONTROL
		0xA4:       "Alt",   // VK_LMENU
		0xA5:       "Alt",   // VK_RMENU
		0xA0:       "Shift", // VK_LSHIFT
		0xA1:       "Shift", // VK_RSHIFT

		// Function keys
		0x70: "F1", 0x71: "F2", 0x72: "F3", 0x73: "F4",
		0x74: "F5", 0x75: "F6", 0x76: "F7", 0x77: "F8",
		0x78: "F9", 0x79: "F10", 0x7A: "F11", 0x7B: "F12",

		// Special keys
		VK_SPACE:  "Space",
		VK_RETURN: "Enter",
		0x09:      "Tab",
		0x1B:      "Esc",
		0x08:      "Backspace",
		0x2E:      "Delete",
		0x24:      "Home",
		0x23:      "End",
		0x21:      "PageUp",
		0x22:      "PageDown",
		0x25:      "Left",
		0x26:      "Up",
		0x27:      "Right",
		0x28:      "Down",

		// Number keys
		0x30: "0", 0x31: "1", 0x32: "2", 0x33: "3", 0x34: "4",
		0x35: "5", 0x36: "6", 0x37: "7", 0x38: "8", 0x39: "9",

		// Letter keys
		0x41: "A", 0x42: "B", 0x43: "C", 0x44: "D", 0x45: "E",
		0x46: "F", 0x47: "G", 0x48: "H", 0x49: "I", 0x4A: "J",
		0x4B: "K", 0x4C: "L", 0x4D: "M", 0x4E: "N", 0x4F: "O",
		0x50: "P", 0x51: "Q", 0x52: "R", 0x53: "S", 0x54: "T",
		0x55: "U", 0x56: "V", 0x57: "W", 0x58: "X", 0x59: "Y",
		0x5A: "Z",
	}

	if name, exists := keyNames[keyCode]; exists {
		return name
	}

	return fmt.Sprintf("Key%d", keyCode)
}

// initializeHotkeyPatterns creates the list of known hotkey patterns
func initializeHotkeyPatterns() []HotkeyPattern {
	return []HotkeyPattern{
		// File operations
		{[]uint32{VK_CONTROL, 0x53}, "Ctrl+S", "Save", false, "File"},
		{[]uint32{VK_CONTROL, 0x4F}, "Ctrl+O", "Open", false, "File"},
		{[]uint32{VK_CONTROL, 0x4E}, "Ctrl+N", "New", false, "File"},
		{[]uint32{VK_CONTROL, 0x50}, "Ctrl+P", "Print", false, "File"},

		// Edit operations
		{[]uint32{VK_CONTROL, 0x43}, "Ctrl+C", "Copy", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x56}, "Ctrl+V", "Paste", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x58}, "Ctrl+X", "Cut", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x5A}, "Ctrl+Z", "Undo", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x59}, "Ctrl+Y", "Redo", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x41}, "Ctrl+A", "Select All", false, "Edit"},
		{[]uint32{VK_CONTROL, 0x46}, "Ctrl+F", "Find", false, "Edit"},

		// Window management
		{[]uint32{VK_MENU, 0x09}, "Alt+Tab", "Switch Window", true, "Window"},
		{[]uint32{VK_MENU, 0x70}, "Alt+F4", "Close Window", false, "Window"},
		{[]uint32{VK_LWIN, 0x44}, "Win+D", "Show Desktop", true, "Window"},
		{[]uint32{VK_LWIN, 0x4C}, "Win+L", "Lock Screen", true, "System"},

		// Browser navigation
		{[]uint32{VK_CONTROL, 0x54}, "Ctrl+T", "New Tab", false, "Browser"},
		{[]uint32{VK_CONTROL, 0x57}, "Ctrl+W", "Close Tab", false, "Browser"},
		{[]uint32{VK_CONTROL, 0x09}, "Ctrl+Tab", "Next Tab", false, "Browser"},
		{[]uint32{VK_CONTROL, VK_SHIFT, 0x09}, "Ctrl+Shift+Tab", "Previous Tab", false, "Browser"},
		{[]uint32{VK_CONTROL, 0x52}, "Ctrl+R", "Refresh", false, "Browser"},
		{[]uint32{VK_CONTROL, 0x4C}, "Ctrl+L", "Address Bar", false, "Browser"},

		// Number shortcuts (Win+1, Win+2, etc.)
		{[]uint32{VK_LWIN, 0x31}, "Win+1", "Launch App 1", true, "Launcher"},
		{[]uint32{VK_LWIN, 0x32}, "Win+2", "Launch App 2", true, "Launcher"},
		{[]uint32{VK_LWIN, 0x33}, "Win+3", "Launch App 3", true, "Launcher"},
		{[]uint32{VK_LWIN, 0x34}, "Win+4", "Launch App 4", true, "Launcher"},
		{[]uint32{VK_LWIN, 0x35}, "Win+5", "Launch App 5", true, "Launcher"},

		// Function keys
		{[]uint32{0x70}, "F1", "Help", false, "Function"},
		{[]uint32{0x73}, "F4", "Address Bar", false, "Function"},
		{[]uint32{0x74}, "F5", "Refresh", false, "Function"},
		{[]uint32{0x7A}, "F11", "Full Screen", false, "Function"},

		// Text formatting
		{[]uint32{VK_CONTROL, 0x42}, "Ctrl+B", "Bold", false, "Format"},
		{[]uint32{VK_CONTROL, 0x49}, "Ctrl+I", "Italic", false, "Format"},
		{[]uint32{VK_CONTROL, 0x55}, "Ctrl+U", "Underline", false, "Format"},

		// Navigation
		{[]uint32{VK_MENU, 0x25}, "Alt+Left", "Back", false, "Navigation"},
		{[]uint32{VK_MENU, 0x27}, "Alt+Right", "Forward", false, "Navigation"},
		{[]uint32{VK_CONTROL, 0x24}, "Ctrl+Home", "Go to Start", false, "Navigation"},
		{[]uint32{VK_CONTROL, 0x23}, "Ctrl+End", "Go to End", false, "Navigation"},

		// System shortcuts
		{[]uint32{VK_CONTROL, VK_SHIFT, 0x1B}, "Ctrl+Shift+Esc", "Task Manager", true, "System"},
		{[]uint32{VK_CONTROL, VK_MENU, 0x2E}, "Ctrl+Alt+Delete", "Security Screen", true, "System"},
		{[]uint32{0x2C}, "PrintScreen", "Screenshot", true, "System"},
	}
}
