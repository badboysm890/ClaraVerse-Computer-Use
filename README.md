# ClaraVerse Computer Use Observer

A comprehensive Windows computer activity recording and workflow analysis tool built in Go. This tool captures detailed user interactions including mouse movements, keyboard input, clipboard operations, application switches, browser navigation, and screenshots to create a complete picture of computer usage workflows.

## 🚀 Features

### Core Recording Capabilities
- **Mouse Events**: Click, drag, scroll, and movement tracking with configurable throttling
- **Keyboard Events**: Key presses, hotkey combinations, and text input completion detection
- **Clipboard Monitoring**: Automatic clipboard content tracking with format detection
- **Application Switching**: Track application focus changes and window transitions
- **Browser Navigation**: Detect tab switches and URL changes across major browsers
- **Screenshot Capture**: Automated screenshots on events or intervals with multiple formats
- **UI Element Detection**: Capture detailed information about interface elements being interacted with

### Advanced Features
- **Text Input Completion**: Intelligent detection of completed text input sessions
- **Drag & Drop Tracking**: Comprehensive drag and drop operation monitoring
- **Text Selection Tracking**: Monitor text selection operations across applications
- **Hotkey Detection**: Global and application-specific hotkey combination tracking
- **Performance Optimization**: Multiple performance modes (Normal, Balanced, Low Energy)
- **Rate Limiting**: Configurable event processing limits to prevent system overload
- **Noise Filtering**: Smart filtering of irrelevant mouse and keyboard events

### Output & Analysis
- **JSON Export**: Structured workflow data in JSON format with timestamps
- **Screenshot Integration**: Base64-encoded screenshots embedded in workflow data
- **Comprehensive Testing**: Built-in test suite for accuracy and performance validation
- **Real-time Monitoring**: Live console output showing captured events

## 📋 Requirements

### System Requirements
- **Operating System**: Windows 10/11 (64-bit)
- **RAM**: Minimum 4GB, Recommended 8GB+
- **Storage**: 100MB+ free space for recordings
- **Permissions**: Administrator privileges recommended for full functionality

### Dependencies
- Go 1.23.4 or later (for building from source)
- Windows API access for system monitoring

## 🛠️ Installation

### Option 1: Download Pre-built Executables
The project includes several pre-built executables for different use cases:

```bash
# Basic UI recorder
./ui_recorder.exe

# Enhanced workflow recorder with all features
./workflow-recorder-enhanced.exe

# Text input focused recorder
./workflow-recorder-with-text-input.exe

# Standard workflow recorder
./workflow-recorder.exe
```

### Option 2: Build from Source

1. **Install Go**: Download and install Go 1.23.4+ from [golang.org](https://golang.org/downloads/)

2. **Clone the repository**:
```bash
git clone https://github.com/your-repo/ClaraVerse-Computer-Use.git
cd ClaraVerse-Computer-Use/Claraverse_observer_windows
```

3. **Install dependencies**:
```bash
go mod download
```

4. **Build the application**:
```bash
# Build main enhanced recorder
go build -o workflow-recorder-enhanced.exe main_enhanced.go

# Or build specific components
go build -o ui_recorder.exe main_enhanced.go workflow_utils.go
```

## 🎯 Quick Start

### Basic Usage

1. **Run the enhanced recorder**:
```bash
./workflow-recorder-enhanced.exe
```

2. **Start performing activities** on your computer - the tool will automatically capture:
   - Mouse movements and clicks
   - Keyboard input
   - Application switches
   - Clipboard operations
   - Screenshots

3. **Stop recording** by pressing `Ctrl+C` in the terminal

4. **Find your recording** in the generated JSON file (e.g., `ui_recording_enhanced_20240101_120000.json`)

### Command Line Options

```bash
# Run with default configuration
./workflow-recorder-enhanced.exe

# Run comprehensive tests
./workflow-recorder-enhanced.exe --test

# Run with specific performance mode
./workflow-recorder-enhanced.exe --performance-mode balanced
```

## ⚙️ Configuration

### Default Configuration
The tool comes with sensible defaults, but can be extensively customized:

```go
WorkflowRecorderConfig{
    RecordMouse:                   true,
    RecordKeyboard:                true,
    CaptureUIElements:             true,
    RecordClipboard:               true,
    RecordHotkeys:                 true,
    RecordTextInputCompletion:     true,
    RecordApplicationSwitches:     true,
    RecordBrowserTabNavigation:    true,
    CaptureScreenshots:            true,
    ScreenshotFormat:              "png",
    ScreenshotJPEGQuality:         85,
    MouseMoveThrottleMs:           100,
    MinDragDistance:               5.0,
    PerformanceMode:               Normal,
}
```

### Performance Modes

#### Normal Mode
- Full feature set enabled
- High accuracy event capture
- Moderate system resource usage

#### Balanced Mode
- Optimized for typical usage
- Smart noise filtering
- Reduced screenshot frequency

#### Low Energy Mode
- Minimal system impact
- Essential events only
- Longer processing intervals

### Screenshot Configuration

```go
// Screenshot settings
CaptureScreenshots:       true,
ScreenshotOnMouseClick:   true,
ScreenshotOnAppSwitch:    true,
ScreenshotOnInterval:     false,
ScreenshotIntervalMs:     5000,
ScreenshotFormat:         "png",    // "png" or "jpeg"
ScreenshotJPEGQuality:    85,       // 1-100 for JPEG
MaxScreenshotWidth:       1920,
MaxScreenshotHeight:      1080,
```

### Filtering and Privacy

```go
// Applications to ignore
IgnoreApplications: []string{
    "dwm.exe", "winlogon.exe", "csrss.exe",
},

// Window titles to ignore
IgnoreWindowTitles: []string{
    "Task Manager", "System Tray",
},

// Focus patterns to ignore
IgnoreFocusPatterns: []string{
    "notification", "tooltip", "popup",
    "1password", "lastpass", "bitwarden",
},
```

## 📊 Output Format

### JSON Structure
The tool generates structured JSON output with the following event types:

```json
{
  "name": "Enhanced Workflow Recording",
  "start_time": 1704067200000,
  "end_time": 1704067800000,
  "events": [
    {
      "event_type": "Click",
      "button": "Left",
      "position": {"x": 500, "y": 300},
      "metadata": {
        "timestamp": 1704067201000,
        "ui_element": {
          "role": "button",
          "name": "Submit",
          "application_name": "notepad.exe"
        }
      }
    }
  ]
}
```

### Event Types

- **MouseEvent**: Click, drag, scroll, movement
- **KeyboardEvent**: Key presses with modifier states
- **ClipboardEvent**: Copy, paste, cut operations
- **ApplicationSwitchEvent**: App focus changes
- **HotkeyEvent**: Hotkey combinations
- **ScreenshotEvent**: Captured images with metadata
- **TextInputCompletedEvent**: Completed text input sessions
- **BrowserTabNavigationEvent**: Browser tab/URL changes
- **DragDropEvent**: Drag and drop operations
- **TextSelectionEvent**: Text selection operations

## 🧪 Testing

### Run Comprehensive Tests

```bash
# Run all tests
go run comprehensive_tests.go

# Run specific test categories
go run comprehensive_tests.go --browser-tests
go run comprehensive_tests.go --performance-tests
go run comprehensive_tests.go --accuracy-tests
```

### Test Categories

1. **Browser Tests**: Automated browser interaction testing
2. **Performance Tests**: CPU, memory, and processing speed validation
3. **Accuracy Tests**: Event capture precision verification

### Test Configuration

```go
TestConfig{
    EnableBrowserTests:     true,
    EnablePerformanceTests: true,
    EnableAccuracyTests:    true,
    TestDurationSeconds:    30,
    MaxEventsPerTest:       1000,
}
```

## 🔧 Advanced Usage

### Integration Example

```go
// Create enhanced recorder
config := getDefaultAdvancedConfig()
recorder, err := NewEnhancedWorkflowRecorder(&config)
if err != nil {
    log.Fatal(err)
}

// Start recording
recorder.StartRecording()

// ... perform activities ...

// Stop and save
recorder.StopRecording()
recorder.SaveWorkflow("my_workflow")
```

### Custom Event Handling

```go
// Handle specific events
recorder.TextInputManager = NewTextInputManager(
    3*time.Second,
    func(event TextInputCompletedEvent) {
        log.Printf("Text completed: %s", event.TextValue)
    },
)
```

### Performance Monitoring

```go
// Get recording statistics
stats := recorder.GetStatistics()
fmt.Printf("Events recorded: %d\n", stats["event_count"])
fmt.Printf("CPU usage: %.2f%%\n", stats["cpu_usage"])
fmt.Printf("Memory usage: %.2f MB\n", stats["memory_usage"])
```

## 🛡️ Privacy and Security

### Data Protection
- Local data storage only (no cloud transmission)
- Configurable filtering of sensitive applications
- Password field exclusion capabilities
- Clipboard content encryption options

### Sensitive Data Handling
- Automatic detection and filtering of password managers
- Exclusion of system security applications
- Configurable anonymization of user data

## 🐛 Troubleshooting

### Common Issues

**High CPU Usage**:
- Switch to "Balanced" or "Low Energy" performance mode
- Increase `MouseMoveThrottleMs` value
- Enable noise filtering options

**Large File Sizes**:
- Disable screenshots or reduce quality
- Enable null value filtering
- Use "compact" serialization mode

**Missing Events**:
- Run as Administrator
- Check application ignore lists
- Verify performance mode settings

**Screenshot Issues**:
- Ensure sufficient disk space
- Check screenshot format settings
- Verify display permissions

### Performance Optimization

```go
// Optimize for speed
config.PerformanceMode = LowEnergy
config.FilterMouseNoise = true
config.FilterKeyboardNoise = true
config.ReduceUIElementCapture = true

// Optimize for quality
config.PerformanceMode = Normal
config.CaptureUIElements = true
config.CaptureScreenshots = true
```

## 📝 File Structure

```
Claraverse_observer_windows/
├── main_enhanced.go              # Main application entry point
├── workflow_utils.go             # Utility functions and helpers
├── integration_example.go        # Usage examples and integration guide
├── comprehensive_tests.go        # Complete test suite
├── optimization_config.go        # Advanced configuration options
├── performance_config.go         # Performance tuning settings
├── text_input_tracker.go         # Text input completion detection
├── text_selection_tracker.go     # Text selection monitoring
├── browser_tracker.go            # Browser navigation tracking
├── hotkey_detector.go            # Hotkey combination detection
├── drag_drop_tracker.go          # Drag and drop monitoring
├── enhanced_clipboard.go         # Advanced clipboard operations
├── advanced_screenshot.go        # Screenshot capture and processing
├── go.mod                        # Go module dependencies
└── *.exe                         # Pre-built executables
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support, issues, or feature requests:
- Open an issue on GitHub
- Check the troubleshooting section above
- Review the comprehensive test results for system compatibility

## 🔄 Updates

The tool automatically logs version information and performance metrics. Check the JSON output for:
- Recording duration and event counts
- Performance statistics
- System compatibility information
- Feature usage analytics
