package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WorkflowRecorderError represents errors from the workflow recorder
type WorkflowRecorderError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

func (e *WorkflowRecorderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewWorkflowError creates a new workflow error
func NewWorkflowError(errorType, message string, cause error) *WorkflowRecorderError {
	return &WorkflowRecorderError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// Common error types
const (
	ErrorTypeConfiguration  = "Configuration"
	ErrorTypeInitialization = "Initialization"
	ErrorTypeRecording      = "Recording"
	ErrorTypeSerialization  = "Serialization"
	ErrorTypeFileIO         = "FileIO"
	ErrorTypeSystem         = "System"
)

// Enhanced utility functions for workflow management

// IsEmptyString checks if a string is empty or contains only null-like values
func IsEmptyString(s *string) bool {
	if s == nil {
		return true
	}

	// Fast path for completely empty strings
	if *s == "" {
		return true
	}

	// Fast path for whitespace-only strings
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return true
	}

	// Check against common null-like values (case-insensitive)
	if len(trimmed) <= 20 { // Reasonable max length for null-like values
		lower := strings.ToLower(trimmed)
		nullValues := []string{
			"null", "nil", "undefined", "(null)", "<null>",
			"n/a", "na", "unknown", "<unknown>", "(unknown)",
			"none", "<none>", "(none)", "empty", "<empty>", "(empty)",
			"bstr()", "variant()", "variant(empty)",
		}

		for _, nullVal := range nullValues {
			if lower == nullVal {
				return true
			}
		}
	}

	return false
}

// FilterEmptyString returns nil if the string is empty or null-like
func FilterEmptyString(s string) *string {
	if IsEmptyString(&s) {
		return nil
	}
	return &s
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-len(suffix)] + suffix
}

// SafeStringPtr safely converts a string to a pointer, returning nil for empty strings
func SafeStringPtr(s string) *string {
	return FilterEmptyString(s)
}

// SafeUint32Ptr returns nil for zero values
func SafeUint32Ptr(n uint32) *uint32 {
	if n == 0 {
		return nil
	}
	return &n
}

// SafeUint64Ptr returns nil for zero values
func SafeUint64Ptr(n uint64) *uint64 {
	if n == 0 {
		return nil
	}
	return &n
}

// Timing utilities

// GetCurrentTimestamp returns the current timestamp in milliseconds since epoch
func GetCurrentTimestamp() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// File and path utilities

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GenerateWorkflowFilename generates a timestamped filename for workflow files
func GenerateWorkflowFilename(name string, format string) string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	safeName := regexp.MustCompile(`[^\w\-_\s]`).ReplaceAllString(name, "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	return fmt.Sprintf("%s_%s.%s", safeName, timestamp, format)
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// JSON utilities

// PrettyPrintJSON formats JSON with indentation
func PrettyPrintJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveJSONToFile saves a struct as JSON to a file
func SaveJSONToFile(v interface{}, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := EnsureDirectoryExists(dir); err != nil {
		return NewWorkflowError(ErrorTypeFileIO, "Failed to create directory", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return NewWorkflowError(ErrorTypeSerialization, "Failed to marshal JSON", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return NewWorkflowError(ErrorTypeFileIO, "Failed to write file", err)
	}

	return nil
}

// LoadJSONFromFile loads JSON from a file into a struct
func LoadJSONFromFile(filename string, v interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return NewWorkflowError(ErrorTypeFileIO, "Failed to read file", err)
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return NewWorkflowError(ErrorTypeSerialization, "Failed to unmarshal JSON", err)
	}

	return nil
}

// String processing utilities

// SanitizeFilename removes or replaces invalid characters for filenames
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := re.ReplaceAllString(filename, "_")

	// Trim spaces and dots from the end
	sanitized = strings.TrimRight(sanitized, " .")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// ExtractNumbers extracts all numbers from a string
func ExtractNumbers(s string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(s, -1)

	var numbers []int
	for _, match := range matches {
		if num, err := strconv.Atoi(match); err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

// ContainsAnyIgnoreCase checks if a string contains any of the provided substrings (case-insensitive)
func ContainsAnyIgnoreCase(s string, substrings []string) bool {
	sLower := strings.ToLower(s)
	for _, substr := range substrings {
		if strings.Contains(sLower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// System information utilities

// GetProcessNameFromPID returns the process name for a given PID (Windows-specific)
func GetProcessNameFromPID(pid uint32) string {
	// This would need to be implemented using Windows APIs
	// For now, return a placeholder
	return fmt.Sprintf("process_%d", pid)
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(s string) bool {
	urlPattern := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlPattern.MatchString(s)
}

// ExtractDomainFromURL extracts the domain from a URL
func ExtractDomainFromURL(url string) string {
	re := regexp.MustCompile(`^https?://([^/]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// Performance utilities

// MeasureExecutionTime measures the execution time of a function
func MeasureExecutionTime(name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	return duration
}

// Rate limiting utilities

// RateLimiter provides simple rate limiting functionality
type RateLimiter struct {
	MaxEvents   int32
	WindowSize  time.Duration
	EventCount  int32
	WindowStart time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxEvents int32, windowSize time.Duration) *RateLimiter {
	return &RateLimiter{
		MaxEvents:   maxEvents,
		WindowSize:  windowSize,
		WindowStart: time.Now(),
	}
}

// Allow checks if an event is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	now := time.Now()

	// Reset window if needed
	if now.Sub(rl.WindowStart) >= rl.WindowSize {
		rl.EventCount = 0
		rl.WindowStart = now
	}

	if rl.EventCount >= rl.MaxEvents {
		return false
	}

	rl.EventCount++
	return true
}

// Configuration validation utilities

// ValidateConfig validates a workflow recorder configuration
func ValidateConfig(config *WorkflowRecorderConfig) error {
	if config == nil {
		return NewWorkflowError(ErrorTypeConfiguration, "Configuration is nil", nil)
	}

	// Validate screenshot settings
	if config.CaptureScreenshots {
		if config.ScreenshotFormat != "png" && config.ScreenshotFormat != "jpeg" {
			return NewWorkflowError(ErrorTypeConfiguration,
				"Invalid screenshot format: must be 'png' or 'jpeg'", nil)
		}

		if config.ScreenshotFormat == "jpeg" && (config.ScreenshotJPEGQuality < 1 || config.ScreenshotJPEGQuality > 100) {
			return NewWorkflowError(ErrorTypeConfiguration,
				"JPEG quality must be between 1 and 100", nil)
		}
	}

	// Validate timeouts and thresholds
	if config.MouseMoveThrottleMs < 0 {
		return NewWorkflowError(ErrorTypeConfiguration,
			"Mouse move throttle cannot be negative", nil)
	}

	if config.MinDragDistance < 0 {
		return NewWorkflowError(ErrorTypeConfiguration,
			"Minimum drag distance cannot be negative", nil)
	}

	return nil
}
