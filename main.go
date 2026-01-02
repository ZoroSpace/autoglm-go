package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"autoglm-go/constants"
	"autoglm-go/phoneagent"
	"autoglm-go/phoneagent/definitions"
	"autoglm-go/phoneagent/helper"
	"autoglm-go/utils"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
	logs "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Config holds all the configuration values from command line arguments
type Config struct {
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	APIKey      string `json:"api_key"`
	MaxSteps    int    `json:"max_steps"`
	DeviceID    string `json:"device_id"`
	Connect     string `json:"connect"`
	Disconnect  string `json:"disconnect"`
	ListDevices bool   `json:"list_devices"`
	EnableTCPIP int    `json:"enable_tcpip"`
	GetDeviceIP string `json:"get_device_ip"`

	WdaUrl     string `json:"wda_url"`
	Pair       bool   `json:"pair"`
	WdaStatus  bool   `json:"wda_status"`
	Quiet      bool   `json:"quiet"`
	ListApps   bool   `json:"list_apps"`
	Lang       string `json:"lang"`
	DeviceType string `json:"device_type"`
	Task       string `json:"task"`
	Debug      bool   `json:"debug"`
}

var rootCmd = &cobra.Command{
	Use:   "main",
	Short: "Phone Agent - AI-powered phone automation",
	Long: `Phone Agent is an AI-powered tool for automating phone operations.
It supports Android devices via ADB, and iOS devices via WebDriverAgent.`,
	Example: `  # Run with default settings (Android)
  go run main.go

  # Specify model endpoint
  go run main.go --base-url http://localhost:8000/v1

  # Use API key for authentication
  go run main.go --apikey sk-xxxxx

  # Run with specific device
  go run main.go --device-id emulator-5554

  # Connect to remote device
  go run main.go --connect 192.168.1.100:5555

  # List connected devices
  go run main.go --list-devices

  # Enable TCP/IP on USB device and get connection info
  go run main.go --enable-tcpip

  # List supported apps
  go run main.go --list-apps

  # iOS specific examples
  # Run with iOS device
  go run main.go --device-type ios "Open Safari and search for iPhone tips"

  # Use WiFi connection for iOS
  go run main.go --device-type ios --wda-url http://192.168.1.100:8100

  # List connected iOS devices
  go run main.go --device-type ios --list-devices

  # Check WebDriverAgent status
  go run main.go --device-type ios --wda-status

  # Pair with iOS device
  go run main.go --device-type ios --pair`,
	Run: func(cmd *cobra.Command, args []string) {
		// Store the task as a space-joined string
		if len(args) > 0 {
			config.Task = strings.Join(args, " ")
		}

		fmt.Printf("Configuration: %s\n", utils.JsonIndent(config))
		fmt.Printf("Task: %s\n", config.Task)
	},
}

var config = &Config{}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as int with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Helper function to get environment variable as float32 with default value
func getEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatValue)
		}
	}
	return defaultValue
}

func init() {
	// Model options
	rootCmd.PersistentFlags().StringVar(&config.BaseURL, "base-url",
		getEnv("PHONE_AGENT_BASE_URL", "https://open.bigmodel.cn/api/paas/v4"),
		"Model API base URL")

	rootCmd.PersistentFlags().StringVar(&config.Model, "model",
		getEnv("PHONE_AGENT_MODEL", "autoglm-phone"),
		"Model name")

	rootCmd.PersistentFlags().StringVar(&config.APIKey, "apikey",
		getEnv("PHONE_AGENT_API_KEY", "EMPTY"),
		"API key for model authentication")

	rootCmd.PersistentFlags().IntVar(&config.MaxSteps, "max-steps",
		getEnvInt("PHONE_AGENT_MAX_STEPS", 100),
		"Maximum steps per task")

	// Device options
	rootCmd.PersistentFlags().StringVarP(&config.DeviceID, "device-id", "d",
		getEnv("PHONE_AGENT_DEVICE_ID", ""),
		"ADB device ID")

	rootCmd.PersistentFlags().StringVarP(&config.Connect, "connect", "c", "",
		"Connect to remote device (e.g., 192.168.1.100:5555)")

	rootCmd.PersistentFlags().StringVar(&config.Disconnect, "disconnect", "",
		"Disconnect from remote device (or 'all' to disconnect all)")

	rootCmd.PersistentFlags().BoolVar(&config.ListDevices, "list-devices", false,
		"List connected devices and exit")

	// For enable-tcpip, we need custom handling to support optional argument
	rootCmd.PersistentFlags().IntVar(&config.EnableTCPIP, "enable-tcpip", 0,
		"Enable TCP/IP debugging on USB device (default port: 5555, use 0 for default)")

	rootCmd.PersistentFlags().StringVar(&config.GetDeviceIP, "get-device-ip", "",
		"Get device IP ")

	// iOS specific options
	rootCmd.PersistentFlags().StringVar(&config.WdaUrl, "wda-url",
		getEnv("PHONE_AGENT_WDA_URL", "http://localhost:8100"),
		"WebDriverAgent URL for iOS (default: http://localhost:8100)")

	rootCmd.PersistentFlags().BoolVar(&config.Pair, "pair", false,
		"Pair with iOS device (required for some operations)")

	rootCmd.PersistentFlags().BoolVar(&config.WdaStatus, "wda-status", false,
		"Show WebDriverAgent status and exit (iOS only)")

	// Other options
	rootCmd.PersistentFlags().BoolVarP(&config.Quiet, "quiet", "q", false,
		"Suppress verbose output")

	rootCmd.PersistentFlags().BoolVar(&config.ListApps, "list-apps", false,
		"List supported apps and exit")

	rootCmd.PersistentFlags().StringVar(&config.Lang,
		"lang",
		getEnv("PHONE_AGENT_LANG", "cn"),
		"Language for system prompt (cn or en, default: cn)")

	rootCmd.PersistentFlags().StringVar(
		&config.DeviceType,
		"device-type",
		"adb",
		"Device type: adb for Android, ios for iPhone (default: adb)",
	)

	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false,
		"Enable debug mode (default: false)")

}

type MessageOnlyFormatter struct{}

func (f *MessageOnlyFormatter) Format(entry *logs.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil // 只返回消息 + 换行符
}

func main() {
	parseArgs()

	logs.SetFormatter(&MessageOnlyFormatter{})
	logs.SetOutput(os.Stdout)

	if config.Debug {
		logs.SetLevel(logs.DebugLevel)
	}

	ctx := context.Background()

	// Handle --list-apps (no system check needed)
	if config.ListApps {
		var supportedApps []string
		if config.DeviceType == constants.IOS {
			logs.Info("Note: For iOS apps, Bundle IDs are configured in: constants/apps.go")
			logs.Info("Supported iOS apps:")
			supportedApps = lo.Keys(constants.APP_PACKAGES_IOS)
		} else {
			logs.Info("Supported Android apps:")
			supportedApps = lo.Keys(constants.APP_PACKAGES_ANDROID)
		}
		sort.Strings(supportedApps)

		for _, app := range supportedApps {
			logs.Infof(" - %s", app)
		}
		return
	}

	device, err := phoneagent.CreateDevice(config.DeviceType)
	if err != nil {
		logs.Errorf("creating device failed, err: %v", err)
		return
	}

	// Handle device commands (these may need partial system checks)
	if hitCmd := handleDeviceCommands(ctx, device); hitCmd {
		return
	}

	if passed := checkSystemRequirements(ctx, config.DeviceType, config.WdaUrl); !passed {
		logs.Info(strings.Repeat("-", 50))
		logs.Error("❌ System check failed. Please fix the issues above.")
		logs.Error("❌ check system requirements failed")
		return
	}

	if passed := checkModelAPI(ctx, config.BaseURL, config.Model, config.APIKey); !passed {
		logs.Error("❌ Model API check failed. Please fix the issues above.")
		logs.Error("❌ check model api failed")
		return
	}

	modelConfig := &definitions.ModelConfig{
		BaseURL:          config.BaseURL,
		ModelName:        config.Model,
		APIKey:           config.APIKey,
		Lang:             config.Lang,
		MaxTokens:        getEnvInt("PHONE_AGENT_MAX_TOKENS", 3000),
		Temperature:      getEnvFloat32("PHONE_AGENT_TEMPERATURE", 0.0),
		TopP:             getEnvFloat32("PHONE_AGENT_TOP_P", 0.85),
		FrequencyPenalty: getEnvFloat32("PHONE_AGENT_FREQUENCY_PENALTY", 0.2),
	}
	agentConfig := &definitions.AgentConfig{
		MaxSteps: config.MaxSteps,
		DeviceID: config.DeviceID,
		Lang:     config.Lang,
		WdaUrl:   config.WdaUrl,
	}

	phoneAgent := phoneagent.NewPhoneAgent(device, modelConfig, agentConfig)

	// Print configuration information
	printConfiguration(ctx, phoneAgent)

	// Run with provided task or enter interactive mode
	if config.Task != "" {
		logs.Infof("Task: %s", config.Task)
		result, err := phoneAgent.Run(ctx, config.Task)
		if err != nil {
			logs.Errorf("Error running task: %v", err)
			return
		}
		logs.Infof("🎉 %s: %s", helper.GetMessage("result", config.Lang), result)
	} else {
		// Interactive mode
		logs.Info("Entering interactive mode. Type 'quit' to exit.")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Enter your task: ")
			task, err := reader.ReadString('\n')
			if err != nil {
				logs.Errorf("Error reading input: %v", err)
				continue
			}

			task = strings.TrimSpace(task)

			// Check for quit commands
			if strings.ToLower(task) == "quit" || strings.ToLower(task) == "exit" || strings.ToLower(task) == "q" {
				logs.Info("Goodbye!")
				break
			}

			// Skip empty tasks
			if task == "" {
				continue
			}

			fmt.Println()
			result, err := phoneAgent.Run(ctx, task)
			if err != nil {
				logs.Errorf("Error: %v", err)
				continue
			}

			logs.Infof("🎉 %s: %s", helper.GetMessage("result", config.Lang), result)

			// Reset agent for next task
			phoneAgent.Reset(ctx)
		}
	}

}

func parseArgs() *Config {
	// Set pre-run validation
	rootCmd.PersistentPreRunE = validateArgs

	// Execute the command
	cobra.CheckErr(rootCmd.Execute())

	return config
}

func validateArgs(cmd *cobra.Command, args []string) error {
	// Validate lang and device-type choices
	if config.Lang != "cn" && config.Lang != "en" {
		return fmt.Errorf("invalid language option: %s. Must be 'cn' or 'en'", config.Lang)
	}

	// only allow adb for now
	if config.DeviceType != "adb" {
		return fmt.Errorf("invalid device type: %s. Must be 'adb'", config.DeviceType)
	}

	return nil
}

func handleDeviceCommands(ctx context.Context, device phoneagent.Device) bool {
	deviceType := config.DeviceType

	// 处理iOS特定命令
	if deviceType == constants.IOS {
		return handleIOSDeviceCommands(ctx)
	}

	// 处理 --list-devices
	if config.ListDevices {
		devices, _ := device.ListDevices(ctx)
		if len(devices) == 0 {
			logs.Info("No devices connected.")
		} else {
			logs.Info("Connected devices:")
			logs.Info(strings.Repeat("-", 60))
			for _, d := range devices {
				statusIcon := "✅"
				if d.Status != "device" {
					statusIcon = "❌"
				}
				connType := d.ConnectionType
				modelInfo := ""
				if d.Model != "" {
					modelInfo = fmt.Sprintf(" (%s)", d.Model)
				}
				logs.Infof("  %s %-30s [%s]%s", statusIcon, d.DeviceID, connType, modelInfo)
			}
		}
		return true
	}

	// 处理 --connect
	if config.Connect != "" {
		logs.Infof("Connecting to %s...", config.Connect)
		message, err := device.Connect(ctx, config.Connect)
		if err != nil {
			logs.Errorf("❌%s", message)
		} else {
			logs.Infof("✅ %s", message)
		}
		return true
	}

	// 处理 --enable-tcpip
	if config.EnableTCPIP > 0 {
		port := config.EnableTCPIP
		logs.Infof("Enabling TCP/IP debugging on port %d...", port)

		err := device.EnableTCPIP(ctx, port, config.DeviceID)
		if err != nil {
			logs.Errorf("❌ %s", "enable tcpip failed")
		} else {
			logs.Infof("✅ %s", "enable tcpip success")
		}
		return true
	}

	// 处理 --get-device-ip
	if len(config.GetDeviceIP) > 0 {
		ip, err := device.GetDeviceIP(ctx, config.GetDeviceIP)
		if err != nil {
			logs.Errorf("❌ %s", "get device ip failed")
		} else {
			logs.Infof("✅ %s: %s", "device ip", ip)
		}
		return true
	}

	// 处理 --disconnect
	if config.Disconnect != "" {
		var (
			message string
			err     error
		)

		if config.Disconnect == "all" {
			logs.Info("Disconnecting all remote devices...")
			message, err = device.Disconnect(ctx, "") // 断开所有连接
		} else {
			logs.Infof("Disconnecting from %s...", config.Disconnect)
			message, err = device.Disconnect(ctx, config.Disconnect)
		}
		logs.Infof("Disconnecting result: %s", message)

		var statusSymbol string
		if err != nil {
			statusSymbol = "❌"
		} else {
			statusSymbol = "✅"
		}
		logs.Infof("%s %s", statusSymbol, message)
		return true
	}

	return false
}

func handleIOSDeviceCommands(ctx context.Context) bool {
	// todo
	return false
}

func checkSystemRequirements(ctx context.Context, deviceType string, wdaURL string) bool {
	logs.Info("🔍 Checking system requirements...")
	logs.Info(strings.Repeat("-", 50))

	// Determine tool name and command
	var toolName, toolCmd string
	if deviceType == constants.ADB {
		toolName = "ADB"
		toolCmd = "adb"
	} else {
		toolName = "libimobiledevice"
		toolCmd = "idevice_id"
	}

	// Check 1: Tool installed
	logs.Infof("1. Checking %s installation... ", toolName)
	_, err := exec.LookPath(toolCmd)
	if err != nil {
		logs.Errorf("❌ FAILED")
		logs.Infof("   Error: %s is not installed or not in PATH.", toolName)
		logs.Infof("   Solution: Install %s:", toolName)
		if deviceType == constants.ADB {
			logs.Infof("     - macOS: brew install android-platform-tools")
			logs.Infof("     - Linux: sudo apt install android-tools-adb")
			logs.Infof("     - Windows: Download from https://developer.android.com/studio/releases/platform-tools")
		} else { // IOS
			logs.Infof("     - macOS: brew install libimobiledevice")
			logs.Infof("     - Linux: sudo apt-get install libimobiledevice-utils")
		}
		return false
	}
	// Double check by running version command
	var versionCmd *exec.Cmd
	if deviceType == constants.ADB {
		versionCmd = exec.Command(toolCmd, "version")
	} else { // IOS
		versionCmd = exec.Command(toolCmd, "-ln")
	}

	output, err := versionCmd.Output()
	if err != nil {
		logs.Errorf("❌ FAILED")
		logs.Infof("   Error: %s command failed to run: %v", toolName, err)
		return false
	}
	lines := strings.Split(string(output), "\n")
	versionLine := ""
	if len(lines) > 0 {
		versionLine = strings.TrimSpace(lines[0])
	}
	if versionLine == "" {
		versionLine = "installed"
	}
	logs.Infof("✅ OK (%s)", versionLine)

	// Check 2: Device connected
	logs.Infof("2. Checking connected devices... ")
	var devices []string
	var deviceIDs []string

	if deviceType == constants.ADB {
		cmd := exec.Command("adb", "devices")
		output, err := cmd.CombinedOutput() // 捕获 stdout + stderr
		if err != nil {
			logs.Errorf("❌ FAILED")
			logs.Infof("   Error: %s command timed out: %v", toolName, err)
			return false
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines[1:] { // 跳过第一行（header）
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, "\tdevice") {
				devices = append(devices, line)
				parts := strings.Split(line, "\t")
				if len(parts) > 0 {
					deviceIDs = append(deviceIDs, parts[0])
				}
			}
		}
	} else { // IOS
		// todo
	}

	if len(devices) == 0 {
		logs.Errorf("❌ FAILED")
		logs.Infof("   Error: No devices connected.")
		logs.Infof("   Solution:")
		if deviceType == constants.ADB {
			logs.Infof("     1. Enable USB debugging on your Android device")
			logs.Infof("     2. Connect via USB and authorize the connection")
			logs.Infof("     3. Or connect remotely: go run main.go --connect <ip>:<port>")
		} else { // IOS
			logs.Infof("     1. Connect your iOS device via USB")
			logs.Infof("     2. Unlock device and tap 'Trust This Computer'")
			logs.Infof("     3. Verify: idevice_id -l")
			logs.Infof("     4. Or connect via WiFi using device IP")
		}
		return false
	}
	displayIDs := deviceIDs
	if len(displayIDs) > 2 {
		displayIDs = deviceIDs[:2]
		displayIDs = append(displayIDs, "...")
	}
	logs.Infof("✅ OK (%d device(s): %s)", len(devices), strings.Join(displayIDs, ", "))

	// Check 3: ADB Keyboard installed (only for ADB) or WebDriverAgent (for iOS)
	if deviceType == constants.ADB {
		logs.Info("3. Checking ADB Keyboard... ")
		cmd := exec.Command("adb", "shell", "ime", "list", "-s")
		output, err := cmd.CombinedOutput()
		if err != nil {
			logs.Error("❌ FAILED")
			logs.Infof("   Error: ADB command timed out: %v", err)
			return false
		}

		imeList := string(output)
		if strings.Contains(imeList, "com.android.adbkeyboard/.AdbIME") {
			logs.Info("✅ OK")
		} else {
			logs.Error("❌ FAILED")
			logs.Infof("   Error: ADB Keyboard is not installed on the device.")
			logs.Infof("   Solution:")
			logs.Infof("     1. Download ADB Keyboard APK from:")
			logs.Infof("        https://github.com/senzhk/ADBKeyBoard/blob/master/ADBKeyboard.apk")
			logs.Infof("     2. Install it on your device: adb install ADBKeyboard.apk")
			logs.Infof("     3. Enable it in Settings > System > Languages & Input > Virtual Keyboard")
			return false
		}

	} else { // IOS
		// todo
	}

	logs.Info(strings.Repeat("-", 50))
	logs.Info("✅ All system checks passed!")

	return true
}

// printConfiguration prints the configuration information
func printConfiguration(ctx context.Context, phoneAgent *phoneagent.PhoneAgent) {

	modelConfig := phoneAgent.ModelConfig
	agentConfig := phoneAgent.AgentConfig
	device := phoneAgent.Device

	logs.Info(strings.Repeat("=", 50))
	if config.DeviceType == constants.ADB {
		logs.Info("Phone Agent - AI-powered phone automation")
	} else {
		logs.Info("Phone Agent iOS - AI-powered iOS automation")
	}

	logs.Info(strings.Repeat("=", 50))
	logs.Infof("Model: %s", modelConfig.ModelName)
	logs.Infof("Base URL: %s", modelConfig.BaseURL)
	logs.Infof("Max Steps: %d", agentConfig.MaxSteps)
	logs.Infof("Language: %s", agentConfig.Lang)
	logs.Infof("Device Type: %s", strings.ToUpper(config.DeviceType))
	if agentConfig.DeviceID != "" {
		logs.Infof("Device: %s", agentConfig.DeviceID)
	}

	devices, err := device.ListDevices(ctx)
	if err != nil {
		logs.Errorf("❌ Failed to list devices, err: %v", err)
		return
	}

	// Show device info
	if len(devices) > 0 {
		d := devices[0]
		if config.DeviceType == constants.IOS {

			deviceName := d.DeviceID
			if len(deviceName) > 16 {
				deviceName = deviceName[:16]
			}
			logs.Infof("Device: %s", deviceName)
			if d.Model != "" {
				logs.Infof("Model: %s", d.Model)
			}
		} else {
			logs.Infof("Device: %s (auto-detected)", d.DeviceID)
		}
	}
	logs.Info(strings.Repeat("=", 50))
}

func checkModelAPI(ctx context.Context, baseURL, modelName, apiKey string) bool {
	logs.Info("🔍 Checking model API...")
	logs.Info(strings.Repeat("-", 50))

	// Check 1: Network connectivity using chat API
	logs.Infof("1. Checking API connectivity (%s)... ", baseURL)

	// Create OpenAI client
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := openai.NewClientWithConfig(config)

	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use chat completion to test connectivity
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "please return hello world",
				},
			},
			MaxCompletionTokens: 5,
			Temperature:         0,
			Stream:              false,
		},
	)
	// Check response
	if err != nil {
		logs.Error("❌ FAILED")
		errorMsg := err.Error()
		// Provide more specific error messages
		switch {
		case strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "connection error"):
			logs.Infof("   Error: Cannot connect to %s", baseURL)
			logs.Info("   Solution:")
			logs.Info("     1. Check if the model server is running")
			logs.Info("     2. Verify the base URL is correct")
			logs.Infof("     3. Try: curl %s/chat/completions", baseURL)
		case strings.Contains(strings.ToLower(errorMsg), "timed out") || strings.Contains(errorMsg, "timeout"):
			logs.Infof("   Error: Connection to %s timed out", baseURL)
			logs.Info("   Solution:")
			logs.Info("     1. Check your network connection")
			logs.Info("     2. Verify the server is responding")
		case strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "name resolution"):
			logs.Info("   Error: Cannot resolve hostname")
			logs.Info("   Solution:")
			logs.Info("     1. Check the URL is correct")
			logs.Info("     2. Verify DNS settings")
		default:
			logs.Infof("   Error: %s", errorMsg)
		}
		return false
	}

	if len(resp.Choices) == 0 {
		logs.Error("❌ FAILED")
		logs.Error("   Error: Received empty response from API")
		return false
	}

	logs.Infof("✅ OK, Response: %s", utils.JsonString(resp))

	logs.Info(strings.Repeat("-", 50))
	logs.Info("✅ Model API checks passed!")

	return true
}
