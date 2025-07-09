package main

import (
	"fmt"
	"time"
)

// Note: PerformanceMode is already defined in main_enhanced.go

// String returns the string representation of PerformanceMode
func (pm PerformanceMode) String() string {
	switch pm {
	case Normal:
		return "Normal"
	case Balanced:
		return "Balanced"
	case LowEnergy:
		return "LowEnergy"
	default:
		return "Unknown"
	}
}

// PerformanceSettings contains computed performance settings based on mode
type PerformanceSettings struct {
	EventProcessingDelayMs   uint64
	MaxEventsPerSecond       *int32
	FilterMouseNoise         bool
	FilterKeyboardNoise      bool
	ReduceUIElementCapture   bool
	MouseMoveThrottleMs      int64
	ScreenshotThrottleMs     uint64
	ClipboardCheckThrottleMs uint64
}

// GetPerformanceSettings returns the appropriate performance settings for a given mode
func GetPerformanceSettings(mode PerformanceMode) PerformanceSettings {
	switch mode {
	case LowEnergy:
		return PerformanceSettings{
			EventProcessingDelayMs:   100, // Very conservative
			MaxEventsPerSecond:       func() *int32 { v := int32(5); return &v }(),
			FilterMouseNoise:         true,
			FilterKeyboardNoise:      true,
			ReduceUIElementCapture:   true,
			MouseMoveThrottleMs:      500,  // Very slow mouse tracking
			ScreenshotThrottleMs:     5000, // Screenshots every 5 seconds max
			ClipboardCheckThrottleMs: 1000, // Check clipboard every 1 second max
		}
	case Balanced:
		return PerformanceSettings{
			EventProcessingDelayMs:   50, // Moderate delays
			MaxEventsPerSecond:       func() *int32 { v := int32(20); return &v }(),
			FilterMouseNoise:         true, // Skip mouse moves/scrolls
			FilterKeyboardNoise:      false,
			ReduceUIElementCapture:   false,
			MouseMoveThrottleMs:      200,  // Moderate mouse tracking
			ScreenshotThrottleMs:     2000, // Screenshots every 2 seconds max
			ClipboardCheckThrottleMs: 500,  // Check clipboard every 0.5 seconds max
		}
	case Normal:
		fallthrough
	default:
		return PerformanceSettings{
			EventProcessingDelayMs:   10,  // Minimal delays
			MaxEventsPerSecond:       nil, // No rate limiting
			FilterMouseNoise:         false,
			FilterKeyboardNoise:      false,
			ReduceUIElementCapture:   false,
			MouseMoveThrottleMs:      100,  // Normal mouse tracking
			ScreenshotThrottleMs:     1000, // Screenshots every 1 second max
			ClipboardCheckThrottleMs: 200,  // Check clipboard every 0.2 seconds max
		}
	}
}

// EnhancedWorkflowRecorderConfig extends the original config with performance management
type EnhancedWorkflowRecorderConfig struct {
	WorkflowRecorderConfig

	// Performance optimization
	PerformanceMode PerformanceMode `json:"performance_mode"`

	// Custom overrides (nil means use performance mode default)
	EventProcessingDelayMs *uint64 `json:"event_processing_delay_ms,omitempty"`
	MaxEventsPerSecond     *int32  `json:"max_events_per_second,omitempty"`

	// Advanced filtering options
	FilterMouseNoise       *bool `json:"filter_mouse_noise,omitempty"`
	FilterKeyboardNoise    *bool `json:"filter_keyboard_noise,omitempty"`
	ReduceUIElementCapture *bool `json:"reduce_ui_element_capture,omitempty"`

	// Threading configuration
	EnableMultithreading bool `json:"enable_multithreading"`

	// Advanced screenshot configuration
	ScreenshotThrottleMs     *uint64 `json:"screenshot_throttle_ms,omitempty"`
	ClipboardCheckThrottleMs *uint64 `json:"clipboard_check_throttle_ms,omitempty"`
}

// NewEnhancedConfig creates a new enhanced configuration with defaults
func NewEnhancedConfig() EnhancedWorkflowRecorderConfig {
	return EnhancedWorkflowRecorderConfig{
		WorkflowRecorderConfig: DefaultConfig(),
		PerformanceMode:        Normal,
		EnableMultithreading:   false, // Apartment threaded by default for better responsiveness
	}
}

// NewLowEnergyConfig creates a configuration optimized for low-end computers
func NewLowEnergyConfig() EnhancedWorkflowRecorderConfig {
	config := NewEnhancedConfig()
	config.PerformanceMode = LowEnergy
	config.RecordTextInputCompletion = false // Disable high-overhead features
	config.CaptureUIElements = false         // Disable expensive UI capture
	config.RecordHotkeys = false             // Simplify processing
	config.CaptureScreenshots = false        // No screenshots for performance
	return config
}

// NewBalancedConfig creates a configuration with balanced performance optimizations
func NewBalancedConfig() EnhancedWorkflowRecorderConfig {
	config := NewEnhancedConfig()
	config.PerformanceMode = Balanced
	config.ScreenshotOnInterval = false // Disable interval screenshots
	return config
}

// GetEffectiveSettings returns the effective performance settings, considering overrides
func (c *EnhancedWorkflowRecorderConfig) GetEffectiveSettings() PerformanceSettings {
	settings := GetPerformanceSettings(c.PerformanceMode)

	// Apply any custom overrides
	if c.EventProcessingDelayMs != nil {
		settings.EventProcessingDelayMs = *c.EventProcessingDelayMs
	}

	if c.MaxEventsPerSecond != nil {
		settings.MaxEventsPerSecond = c.MaxEventsPerSecond
	}

	if c.FilterMouseNoise != nil {
		settings.FilterMouseNoise = *c.FilterMouseNoise
	}

	if c.FilterKeyboardNoise != nil {
		settings.FilterKeyboardNoise = *c.FilterKeyboardNoise
	}

	if c.ReduceUIElementCapture != nil {
		settings.ReduceUIElementCapture = *c.ReduceUIElementCapture
	}

	if c.ScreenshotThrottleMs != nil {
		settings.ScreenshotThrottleMs = *c.ScreenshotThrottleMs
	}

	if c.ClipboardCheckThrottleMs != nil {
		settings.ClipboardCheckThrottleMs = *c.ClipboardCheckThrottleMs
	}

	// Apply mouse move throttle from base config
	settings.MouseMoveThrottleMs = c.MouseMoveThrottleMs

	return settings
}

// ShouldFilterEvent determines if an event should be filtered based on performance settings
func (c *EnhancedWorkflowRecorderConfig) ShouldFilterEvent(event interface{}) bool {
	settings := c.GetEffectiveSettings()

	switch e := event.(type) {
	case *MouseEvent:
		if settings.FilterMouseNoise {
			// Filter mouse moves and wheel events but keep clicks
			if e.EventType == MouseMove || e.EventType == MouseWheel {
				return true
			}
		}

	case *KeyboardEvent:
		if settings.FilterKeyboardNoise {
			// Filter key-down events and non-printable keys
			if e.IsKeyDown || e.Character == nil {
				return true
			}
		}
	}

	return false
}

// GetEventDelay returns the appropriate delay between event processing cycles
func (c *EnhancedWorkflowRecorderConfig) GetEventDelay() time.Duration {
	settings := c.GetEffectiveSettings()
	return time.Duration(settings.EventProcessingDelayMs) * time.Millisecond
}

// CreateRateLimiter creates a rate limiter based on the configuration
func (c *EnhancedWorkflowRecorderConfig) CreateRateLimiter() *RateLimiter {
	settings := c.GetEffectiveSettings()

	if settings.MaxEventsPerSecond != nil {
		return NewRateLimiter(*settings.MaxEventsPerSecond, time.Second)
	}

	return nil // No rate limiting
}

// ValidateEnhancedConfig validates an enhanced configuration
func ValidateEnhancedConfig(config *EnhancedWorkflowRecorderConfig) error {
	if config == nil {
		return NewWorkflowError(ErrorTypeConfiguration, "Enhanced configuration is nil", nil)
	}

	// Validate base configuration
	if err := ValidateConfig(&config.WorkflowRecorderConfig); err != nil {
		return err
	}

	// Validate performance mode
	if config.PerformanceMode < Normal || config.PerformanceMode > LowEnergy {
		return NewWorkflowError(ErrorTypeConfiguration, "Invalid performance mode", nil)
	}

	// Validate custom overrides
	if config.EventProcessingDelayMs != nil && *config.EventProcessingDelayMs > 10000 {
		return NewWorkflowError(ErrorTypeConfiguration,
			"Event processing delay too high (max 10 seconds)", nil)
	}

	if config.MaxEventsPerSecond != nil && (*config.MaxEventsPerSecond < 1 || *config.MaxEventsPerSecond > 1000) {
		return NewWorkflowError(ErrorTypeConfiguration,
			"Max events per second must be between 1 and 1000", nil)
	}

	return nil
}

// OptimizeForSystem automatically adjusts configuration based on system capabilities
func (c *EnhancedWorkflowRecorderConfig) OptimizeForSystem() {
	// This is a placeholder for system-specific optimizations
	// In a real implementation, you would:
	// 1. Check CPU cores and speed
	// 2. Check available RAM
	// 3. Check system load
	// 4. Adjust performance mode accordingly

	// For now, we'll provide some basic heuristics

	// If low-energy mode is already set, don't change it
	if c.PerformanceMode == LowEnergy {
		return
	}

	// You could implement actual system detection here
	// For example, using Windows APIs to check system specs
}

// LogPerformanceSettings logs the current performance configuration
func (c *EnhancedWorkflowRecorderConfig) LogPerformanceSettings() {
	settings := c.GetEffectiveSettings()

	fmt.Printf("Performance Configuration:\n")
	fmt.Printf("  Mode: %s\n", c.PerformanceMode)
	fmt.Printf("  Event Delay: %dms\n", settings.EventProcessingDelayMs)

	if settings.MaxEventsPerSecond != nil {
		fmt.Printf("  Max Events/Second: %d\n", *settings.MaxEventsPerSecond)
	} else {
		fmt.Printf("  Max Events/Second: unlimited\n")
	}

	fmt.Printf("  Filter Mouse Noise: %t\n", settings.FilterMouseNoise)
	fmt.Printf("  Filter Keyboard Noise: %t\n", settings.FilterKeyboardNoise)
	fmt.Printf("  Reduce UI Capture: %t\n", settings.ReduceUIElementCapture)
	fmt.Printf("  Mouse Throttle: %dms\n", settings.MouseMoveThrottleMs)
	fmt.Printf("  Screenshot Throttle: %dms\n", settings.ScreenshotThrottleMs)
	fmt.Printf("  Clipboard Throttle: %dms\n", settings.ClipboardCheckThrottleMs)
}
