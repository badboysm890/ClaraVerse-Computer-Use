package main

import (
	"encoding/json"
	"strings"
)

// Advanced configuration extensions
type AdvancedWorkflowConfig struct {
	// Serialization optimization
	EnableDualSerialization  bool   `json:"enable_dual_serialization"`
	EnableNullValueFiltering bool   `json:"enable_null_value_filtering"`
	EnableOmitEmptyFields    bool   `json:"enable_omit_empty_fields"`
	SerializationMode        string `json:"serialization_mode"` // "compact", "readable", "minimal"

	// Advanced screenshot options
	AdvancedScreenshotMode     bool   `json:"advanced_screenshot_mode"`
	ScreenshotCompressionLevel int    `json:"screenshot_compression_level"` // 1-9
	AutoScaleScreenshots       bool   `json:"auto_scale_screenshots"`
	ScreenshotWatermark        string `json:"screenshot_watermark"`

	// Enhanced clipboard options
	ClipboardFormatPriority    []string `json:"clipboard_format_priority"`
	ClipboardSizeTracking      bool     `json:"clipboard_size_tracking"`
	ClipboardContentValidation bool     `json:"clipboard_content_validation"`
	ClipboardEncryptionKey     string   `json:"clipboard_encryption_key,omitempty"`

	// Performance tuning
	PerformanceProfile string `json:"performance_profile"` // "speed", "quality", "balanced"
	EventBufferSize    int    `json:"event_buffer_size"`
	AsyncProcessing    bool   `json:"async_processing"`
	MemoryOptimization bool   `json:"memory_optimization"`

	// Browser detection enhancements
	BrowserSpecificSettings map[string]BrowserConfig `json:"browser_specific_settings"`
	AutoDetectBrowserTabs   bool                     `json:"auto_detect_browser_tabs"`
	TrackBrowserBookmarks   bool                     `json:"track_browser_bookmarks"`

	// Security and privacy
	DataEncryption          bool `json:"data_encryption"`
	AnonymizeUserData       bool `json:"anonymize_user_data"`
	ExcludePasswordFields   bool `json:"exclude_password_fields"`
	SecureClipboardHandling bool `json:"secure_clipboard_handling"`

	// Testing and validation
	EnableTestingMode bool   `json:"enable_testing_mode"`
	ValidationLevel   string `json:"validation_level"` // "basic", "strict", "paranoid"
	AutoRunTests      bool   `json:"auto_run_tests"`
	TestReportFormat  string `json:"test_report_format"` // "json", "html", "xml"
}

// Browser-specific configuration
type BrowserConfig struct {
	DetectionTimeout    int      `json:"detection_timeout_ms"`
	IgnorePatterns      []string `json:"ignore_patterns"`
	SpecialHandling     bool     `json:"special_handling"`
	JavaScriptExecution bool     `json:"javascript_execution"`
}

// Null value filtering patterns
var advancedNullPatterns = []string{
	"null", "undefined", "unknown", "", "n/a", "none", "empty", "void",
	"blank", "na", "nil", "nothing", "no data", "no content",
	"not available", "not specified", "unspecified", "default",
	"placeholder", "sample", "dummy", "test", "lorem ipsum",
	"[object Object]", "{}", "[]", "0", "false", "NaN",
}

// Get default advanced configuration
func getDefaultAdvancedConfig() AdvancedWorkflowConfig {
	return AdvancedWorkflowConfig{
		// Serialization optimization
		EnableDualSerialization:  true,
		EnableNullValueFiltering: true,
		EnableOmitEmptyFields:    true,
		SerializationMode:        "compact",

		// Advanced screenshot options
		AdvancedScreenshotMode:     true,
		ScreenshotCompressionLevel: 6,
		AutoScaleScreenshots:       true,
		ScreenshotWatermark:        "",

		// Enhanced clipboard options
		ClipboardFormatPriority:    []string{"html", "rtf", "unicode", "text"},
		ClipboardSizeTracking:      true,
		ClipboardContentValidation: true,
		ClipboardEncryptionKey:     "",

		// Performance tuning
		PerformanceProfile: "balanced",
		EventBufferSize:    1000,
		AsyncProcessing:    true,
		MemoryOptimization: true,

		// Browser detection enhancements
		BrowserSpecificSettings: map[string]BrowserConfig{
			"chrome": {
				DetectionTimeout:    5000,
				IgnorePatterns:      []string{"chrome-extension://", "chrome://"},
				SpecialHandling:     true,
				JavaScriptExecution: false,
			},
			"firefox": {
				DetectionTimeout:    5000,
				IgnorePatterns:      []string{"moz-extension://", "about:"},
				SpecialHandling:     true,
				JavaScriptExecution: false,
			},
			"edge": {
				DetectionTimeout:    5000,
				IgnorePatterns:      []string{"edge://", "extension://"},
				SpecialHandling:     true,
				JavaScriptExecution: false,
			},
		},
		AutoDetectBrowserTabs: true,
		TrackBrowserBookmarks: false,

		// Security and privacy
		DataEncryption:          false,
		AnonymizeUserData:       false,
		ExcludePasswordFields:   true,
		SecureClipboardHandling: true,

		// Testing and validation
		EnableTestingMode: false,
		ValidationLevel:   "basic",
		AutoRunTests:      false,
		TestReportFormat:  "json",
	}
}

// Apply advanced configuration to existing config
func applyAdvancedConfig(baseConfig WorkflowRecorderConfig, advancedConfig AdvancedWorkflowConfig) WorkflowRecorderConfig {
	// Performance profile adjustments
	switch advancedConfig.PerformanceProfile {
	case "speed":
		baseConfig.PerformanceMode = 0 // Assuming 0 is fastest
		if baseConfig.EventProcessingDelayMs != nil {
			*baseConfig.EventProcessingDelayMs = 1
		}
		baseConfig.FilterMouseNoise = false
		baseConfig.FilterKeyboardNoise = false
	case "quality":
		baseConfig.PerformanceMode = 2 // Assuming 2 is highest quality
		if baseConfig.EventProcessingDelayMs != nil {
			*baseConfig.EventProcessingDelayMs = 10
		}
		baseConfig.FilterMouseNoise = true
		baseConfig.FilterKeyboardNoise = true
		baseConfig.CaptureUIElements = true
	case "balanced":
		baseConfig.PerformanceMode = 1 // Assuming 1 is balanced
		if baseConfig.EventProcessingDelayMs != nil {
			*baseConfig.EventProcessingDelayMs = 5
		}
	}

	// Screenshot enhancements
	if advancedConfig.AdvancedScreenshotMode {
		baseConfig.CaptureScreenshots = true
		baseConfig.ScreenshotJPEGQuality = min(max(advancedConfig.ScreenshotCompressionLevel*10, 10), 90)
	}

	// Browser specific timeouts
	if browserConfig, exists := advancedConfig.BrowserSpecificSettings["chrome"]; exists {
		baseConfig.BrowserDetectionTimeoutMs = int64(browserConfig.DetectionTimeout)
	}

	return baseConfig
}

// Serialize event with advanced options
func serializeEventAdvanced(event WorkflowEvent, config AdvancedWorkflowConfig) ([]byte, error) {
	if config.EnableDualSerialization {
		return serializeEventDual(event, config)
	}

	// Standard JSON serialization with optimizations
	if config.EnableNullValueFiltering {
		event = filterNullValues(event)
	}

	if config.SerializationMode == "compact" {
		return json.Marshal(event)
	} else if config.SerializationMode == "readable" {
		return json.MarshalIndent(event, "", "  ")
	} else if config.SerializationMode == "minimal" {
		return serializeMinimal(event)
	}

	return json.Marshal(event)
}

// Dual serialization for internal vs external use
func serializeEventDual(event WorkflowEvent, config AdvancedWorkflowConfig) ([]byte, error) {
	// Create internal representation (full data)
	internalEvent := event

	// Create external representation (filtered data)
	externalEvent := event
	if config.EnableNullValueFiltering {
		externalEvent = filterNullValues(externalEvent)
	}
	if config.AnonymizeUserData {
		externalEvent = anonymizeEvent(externalEvent)
	}

	// For now, return external representation
	// In a full implementation, both would be stored/used appropriately
	return json.Marshal(externalEvent)
}

// Filter null values from event
func filterNullValues(event WorkflowEvent) WorkflowEvent {
	// This is a simplified implementation
	// In reality, you'd use reflection or type-specific filtering

	switch e := event.(type) {
	case *ClipboardEvent:
		if isAdvancedNullValue(e.Content) {
			return nil // Filter out null events
		}
		return e
	case *KeyboardEvent:
		if e.Character != nil && isAdvancedNullValue(*e.Character) {
			e.Character = nil
		}
		return e
	default:
		return event
	}
}

// Check if value should be considered null/empty
func isAdvancedNullValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))

	for _, pattern := range advancedNullPatterns {
		if normalized == pattern {
			return true
		}
	}

	return false
}

// Anonymize sensitive data in events
func anonymizeEvent(event WorkflowEvent) WorkflowEvent {
	switch e := event.(type) {
	case *ClipboardEvent:
		if len(e.Content) > 10 {
			e.Content = e.Content[:3] + "***" + e.Content[len(e.Content)-3:]
		}
		return e
	case *KeyboardEvent:
		if e.Character != nil {
			masked := "***"
			e.Character = &masked
		}
		return e
	default:
		return event
	}
}

// Minimal serialization for size optimization
func serializeMinimal(event WorkflowEvent) ([]byte, error) {
	// Create a minimal representation with only essential fields
	switch e := event.(type) {
	case *MouseEvent:
		minimal := map[string]interface{}{
			"type": "mouse",
			"evt":  e.EventType,
			"btn":  e.Button,
			"x":    e.Position.X,
			"y":    e.Position.Y,
			"ts":   e.Metadata.Timestamp,
		}
		return json.Marshal(minimal)
	case *KeyboardEvent:
		minimal := map[string]interface{}{
			"type": "key",
			"code": e.KeyCode,
			"down": e.IsKeyDown,
			"ts":   e.Metadata.Timestamp,
		}
		return json.Marshal(minimal)
	case *ClipboardEvent:
		minimal := map[string]interface{}{
			"type": "clip",
			"act":  e.Action,
			"size": e.ContentSize,
			"ts":   e.Metadata.Timestamp,
		}
		return json.Marshal(minimal)
	default:
		// Fallback to standard serialization
		return json.Marshal(event)
	}
}

// Validate configuration settings
func validateAdvancedConfig(config AdvancedWorkflowConfig) []string {
	var errors []string

	// Validate performance profile
	validProfiles := []string{"speed", "quality", "balanced"}
	if !contains(validProfiles, config.PerformanceProfile) {
		errors = append(errors, "Invalid performance profile: "+config.PerformanceProfile)
	}

	// Validate serialization mode
	validModes := []string{"compact", "readable", "minimal"}
	if !contains(validModes, config.SerializationMode) {
		errors = append(errors, "Invalid serialization mode: "+config.SerializationMode)
	}

	// Validate validation level
	validLevels := []string{"basic", "strict", "paranoid"}
	if !contains(validLevels, config.ValidationLevel) {
		errors = append(errors, "Invalid validation level: "+config.ValidationLevel)
	}

	// Validate compression level
	if config.ScreenshotCompressionLevel < 1 || config.ScreenshotCompressionLevel > 9 {
		errors = append(errors, "Screenshot compression level must be between 1 and 9")
	}

	// Validate event buffer size
	if config.EventBufferSize < 100 || config.EventBufferSize > 10000 {
		errors = append(errors, "Event buffer size must be between 100 and 10000")
	}

	return errors
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Load advanced configuration from file
func loadAdvancedConfig(filename string) (AdvancedWorkflowConfig, error) {
	config := getDefaultAdvancedConfig()

	// In a real implementation, this would read from a file
	// For now, return default config
	return config, nil
}

// Save advanced configuration to file
func saveAdvancedConfig(config AdvancedWorkflowConfig, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// In a real implementation, this would write to a file
	// For now, just validate the JSON is correct
	_ = data
	return nil
}

// Get optimized configuration based on system resources
func getOptimizedConfig() AdvancedWorkflowConfig {
	config := getDefaultAdvancedConfig()

	// Adjust based on available memory (simplified)
	// In reality, you'd check actual system resources
	config.MemoryOptimization = true
	config.EventBufferSize = 500
	config.PerformanceProfile = "balanced"

	return config
}
