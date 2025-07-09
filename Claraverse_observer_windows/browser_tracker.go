package main

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TabAction represents browser tab actions
type TabAction string

const (
	TabCreated    TabAction = "Created"
	TabSwitched   TabAction = "Switched"
	TabClosed     TabAction = "Closed"
	TabMoved      TabAction = "Moved"
	TabDuplicated TabAction = "Duplicated"
	TabPinned     TabAction = "Pinned"
	TabRefreshed  TabAction = "Refreshed"
)

// TabNavigationMethod represents how tab navigation occurred
type TabNavigationMethod string

const (
	TabNavigationKeyboardShortcut TabNavigationMethod = "KeyboardShortcut"
	TabNavigationTabClick         TabNavigationMethod = "TabClick"
	TabNavigationNewTabButton     TabNavigationMethod = "NewTabButton"
	TabNavigationCloseButton      TabNavigationMethod = "CloseButton"
	TabNavigationContextMenu      TabNavigationMethod = "ContextMenu"
	TabNavigationAddressBar       TabNavigationMethod = "AddressBar"
	TabNavigationLinkNewTab       TabNavigationMethod = "LinkNewTab"
	TabNavigationOther            TabNavigationMethod = "Other"
)

// BrowserTabNavigationEvent represents browser tab operations
type BrowserTabNavigationEvent struct {
	Action          TabAction           `json:"action"`
	Method          TabNavigationMethod `json:"method"`
	ToURL           string              `json:"to_url,omitempty"`
	FromURL         string              `json:"from_url,omitempty"`
	ToTitle         string              `json:"to_title,omitempty"`
	FromTitle       string              `json:"from_title,omitempty"`
	Browser         string              `json:"browser"`
	TabIndex        uint32              `json:"tab_index,omitempty"`
	TotalTabs       uint32              `json:"total_tabs,omitempty"`
	PageDwellTimeMs uint64              `json:"page_dwell_time_ms,omitempty"`
	IsBackForward   bool                `json:"is_back_forward"`
	Metadata        EventMetadata       `json:"metadata"`
}

// BrowserState tracks the state of a browser window
type BrowserState struct {
	ProcessID     uint32
	WindowTitle   string
	CurrentURL    string
	LastURLChange time.Time
	TabCount      uint32
	LastTabAction time.Time
	RecentHotkeys []string
	RecentClicks  []Position
}

// BrowserTabTracker tracks browser tab navigation
type BrowserTabTracker struct {
	BrowserStates   map[uint32]*BrowserState // ProcessID -> BrowserState
	LastNavigation  time.Time
	EventCallback   func(BrowserTabNavigationEvent)
	URLPatterns     map[string]*regexp.Regexp
	BrowserPatterns map[string]*regexp.Regexp
	Mutex           sync.RWMutex
}

// NewBrowserTabTracker creates a new browser tab tracker
func NewBrowserTabTracker(callback func(BrowserTabNavigationEvent)) *BrowserTabTracker {
	tracker := &BrowserTabTracker{
		BrowserStates:   make(map[uint32]*BrowserState),
		EventCallback:   callback,
		URLPatterns:     make(map[string]*regexp.Regexp),
		BrowserPatterns: make(map[string]*regexp.Regexp),
	}

	// Initialize URL extraction patterns
	tracker.initializePatterns()

	return tracker
}

// HandleWindowChange processes window focus changes to detect browser navigation
func (btt *BrowserTabTracker) HandleWindowChange(element *UIElement) {
	if element == nil || !btt.isBrowserWindow(element) {
		return
	}

	btt.Mutex.Lock()
	defer btt.Mutex.Unlock()

	processID := element.ProcessID
	windowTitle := element.WindowTitle
	currentURL := btt.extractURL(windowTitle)

	// Get or create browser state
	browserState, exists := btt.BrowserStates[processID]
	if !exists {
		browserState = &BrowserState{
			ProcessID:     processID,
			WindowTitle:   windowTitle,
			CurrentURL:    currentURL,
			LastURLChange: time.Now(),
			TabCount:      1,
		}
		btt.BrowserStates[processID] = browserState
		return // First time seeing this browser, don't emit event
	}

	// Check for URL change (tab navigation)
	if currentURL != "" && currentURL != browserState.CurrentURL {
		dwellTime := uint64(time.Since(browserState.LastURLChange).Milliseconds())

		event := BrowserTabNavigationEvent{
			Action:          TabSwitched,
			Method:          btt.determineNavigationMethod(browserState),
			ToURL:           currentURL,
			FromURL:         browserState.CurrentURL,
			ToTitle:         btt.extractTitle(windowTitle),
			FromTitle:       btt.extractTitle(browserState.WindowTitle),
			Browser:         btt.getBrowserName(element.ApplicationName),
			PageDwellTimeMs: dwellTime,
			IsBackForward:   btt.isBackForwardNavigation(browserState.CurrentURL, currentURL),
			Metadata:        createEventMetadata(),
		}

		// Set UI element in metadata
		event.Metadata.UIElement = element

		// Update browser state
		browserState.CurrentURL = currentURL
		browserState.WindowTitle = windowTitle
		browserState.LastURLChange = time.Now()
		browserState.LastTabAction = time.Now()

		// Emit event
		if btt.EventCallback != nil {
			go btt.EventCallback(event)
		}

		log.Printf("Browser navigation detected: %s -> %s (%s)",
			event.FromURL, event.ToURL, event.Browser)
	}

	// Update last seen state
	browserState.WindowTitle = windowTitle
	if currentURL != "" {
		browserState.CurrentURL = currentURL
	}
}

// HandleHotkey processes hotkey events that might indicate browser navigation
func (btt *BrowserTabTracker) HandleHotkey(combination string, activeElement *UIElement) {
	if activeElement == nil || !btt.isBrowserWindow(activeElement) {
		return
	}

	// Check for browser navigation hotkeys
	if btt.isBrowserNavigationHotkey(combination) {
		btt.Mutex.Lock()
		defer btt.Mutex.Unlock()

		processID := activeElement.ProcessID
		if browserState, exists := btt.BrowserStates[processID]; exists {
			browserState.RecentHotkeys = append(browserState.RecentHotkeys, combination)
			// Keep only recent hotkeys (last 5)
			if len(browserState.RecentHotkeys) > 5 {
				browserState.RecentHotkeys = browserState.RecentHotkeys[1:]
			}
		}
	}
}

// HandleClick processes mouse clicks that might indicate browser navigation
func (btt *BrowserTabTracker) HandleClick(position Position, activeElement *UIElement) {
	if activeElement == nil || !btt.isBrowserWindow(activeElement) {
		return
	}

	btt.Mutex.Lock()
	defer btt.Mutex.Unlock()

	processID := activeElement.ProcessID
	if browserState, exists := btt.BrowserStates[processID]; exists {
		browserState.RecentClicks = append(browserState.RecentClicks, position)
		// Keep only recent clicks (last 10)
		if len(browserState.RecentClicks) > 10 {
			browserState.RecentClicks = browserState.RecentClicks[1:]
		}
	}
}

// Helper methods
func (btt *BrowserTabTracker) initializePatterns() {
	// URL extraction patterns for different browsers
	urlPatterns := map[string]string{
		"chrome":  `https?://[^\s\-—]+`,
		"firefox": `https?://[^\s\-—]+`,
		"edge":    `https?://[^\s\-—]+`,
		"safari":  `https?://[^\s\-—]+`,
	}

	for browser, pattern := range urlPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			btt.URLPatterns[browser] = compiled
		}
	}

	// Browser name patterns
	browserPatterns := map[string]string{
		"chrome":  `(?i)chrome|chromium`,
		"firefox": `(?i)firefox|mozilla`,
		"edge":    `(?i)edge|msedge`,
		"safari":  `(?i)safari`,
		"opera":   `(?i)opera`,
	}

	for browser, pattern := range browserPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			btt.BrowserPatterns[browser] = compiled
		}
	}
}

func (btt *BrowserTabTracker) isBrowserWindow(element *UIElement) bool {
	if element == nil {
		return false
	}

	appName := strings.ToLower(element.ApplicationName)
	windowTitle := strings.ToLower(element.WindowTitle)

	browsers := []string{"chrome", "firefox", "edge", "safari", "opera", "brave", "vivaldi"}

	for _, browser := range browsers {
		if strings.Contains(appName, browser) || strings.Contains(windowTitle, browser) {
			return true
		}
	}

	return false
}

func (btt *BrowserTabTracker) extractURL(windowTitle string) string {
	if windowTitle == "" {
		return ""
	}

	// Try different URL extraction patterns
	for _, pattern := range btt.URLPatterns {
		if matches := pattern.FindString(windowTitle); matches != "" {
			return matches
		}
	}

	// Fallback: look for http/https patterns
	urlPattern := regexp.MustCompile(`https?://[^\s\-—\|\[\]]+`)
	if match := urlPattern.FindString(windowTitle); match != "" {
		return strings.TrimRight(match, ".,;)")
	}

	return ""
}

func (btt *BrowserTabTracker) extractTitle(windowTitle string) string {
	if windowTitle == "" {
		return ""
	}

	// Remove URL from title
	title := windowTitle
	if url := btt.extractURL(windowTitle); url != "" {
		title = strings.Replace(title, url, "", 1)
	}

	// Remove browser name suffixes
	for browser := range btt.BrowserPatterns {
		browserName := " - " + strings.Title(browser)
		title = strings.Replace(title, browserName, "", -1)
	}

	// Clean up the title
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "-—|[](){}")
	title = strings.TrimSpace(title)

	return title
}

func (btt *BrowserTabTracker) getBrowserName(appName string) string {
	if appName == "" {
		return "Unknown"
	}

	appNameLower := strings.ToLower(appName)

	for browser, pattern := range btt.BrowserPatterns {
		if pattern.MatchString(appNameLower) {
			return strings.Title(browser)
		}
	}

	return appName
}

func (btt *BrowserTabTracker) determineNavigationMethod(browserState *BrowserState) TabNavigationMethod {
	// Check recent hotkeys for navigation patterns
	for _, hotkey := range browserState.RecentHotkeys {
		switch hotkey {
		case "Ctrl+T", "Ctrl+Shift+T":
			return TabNavigationNewTabButton
		case "Ctrl+W", "Ctrl+F4":
			return TabNavigationCloseButton
		case "Ctrl+Tab", "Ctrl+Shift+Tab", "Ctrl+1", "Ctrl+2", "Ctrl+3", "Ctrl+4", "Ctrl+5", "Ctrl+6", "Ctrl+7", "Ctrl+8", "Ctrl+9":
			return TabNavigationKeyboardShortcut
		case "Ctrl+L", "F6":
			return TabNavigationAddressBar
		}
	}

	// If there were recent clicks, likely tab click
	if len(browserState.RecentClicks) > 0 {
		return TabNavigationTabClick
	}

	return TabNavigationOther
}

func (btt *BrowserTabTracker) isBackForwardNavigation(fromURL, toURL string) bool {
	// Simple heuristic: if URLs share a domain and one is shorter/longer, might be back/forward
	if fromURL == "" || toURL == "" {
		return false
	}

	// Extract domains
	fromDomain := btt.extractDomain(fromURL)
	toDomain := btt.extractDomain(toURL)

	// Same domain with different paths might indicate back/forward
	return fromDomain == toDomain && fromDomain != ""
}

func (btt *BrowserTabTracker) extractDomain(url string) string {
	domainPattern := regexp.MustCompile(`https?://([^/]+)`)
	if matches := domainPattern.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (btt *BrowserTabTracker) isBrowserNavigationHotkey(combination string) bool {
	navigationHotkeys := []string{
		"Ctrl+T", "Ctrl+Shift+T", "Ctrl+W", "Ctrl+F4",
		"Ctrl+Tab", "Ctrl+Shift+Tab", "Ctrl+L", "F6",
		"Ctrl+1", "Ctrl+2", "Ctrl+3", "Ctrl+4", "Ctrl+5",
		"Ctrl+6", "Ctrl+7", "Ctrl+8", "Ctrl+9",
		"Ctrl+R", "F5", "Ctrl+F5", "Ctrl+Shift+R",
		"Alt+Left", "Alt+Right", "Backspace",
	}

	for _, hotkey := range navigationHotkeys {
		if combination == hotkey {
			return true
		}
	}

	return false
}
