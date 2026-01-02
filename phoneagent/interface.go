package phoneagent

import (
	"context"
	"fmt"

	"autoglm-go/constants"
	"autoglm-go/phoneagent/android"
	"autoglm-go/phoneagent/definitions"
	"autoglm-go/phoneagent/ios"
)

// DeviceOperator 定义设备操作接口
type DeviceOperator interface {
	GetScreenshot(ctx context.Context, deviceID string) (*definitions.Screenshot, error)
	GetCurrentApp(ctx context.Context, deviceID string) (string, error)
	Tap(ctx context.Context, x, y int, deviceID string) error
	DoubleTap(ctx context.Context, x, y int, deviceID string) error
	LongPress(ctx context.Context, x, y int, deviceID string) error
	Swipe(ctx context.Context, startX, startY, endX, endY int, deviceID string) error
	Back(ctx context.Context, deviceID string) error
	Home(ctx context.Context, deviceID string) error
	LaunchApp(ctx context.Context, appName, deviceID string) (bool, error)
	TypeText(ctx context.Context, text, deviceID string) error
	ClearText(ctx context.Context, deviceID string) error
	DetectAndSetADBKeyboard(ctx context.Context, deviceID string) (string, error)
	RestoreKeyboard(ctx context.Context, ime, deviceID string) error
}

// DeviceManager 管理设备连接和状态
type DeviceManager interface {
	Connect(ctx context.Context, address string) (string, error)
	Disconnect(ctx context.Context, address string) (string, error)
	ListDevices(ctx context.Context) ([]definitions.DeviceInfo, error)
	GetDeviceInfo(ctx context.Context, deviceID string) (*definitions.DeviceInfo, error)
	IsConnected(ctx context.Context, deviceID string) bool
	EnableTCPIP(ctx context.Context, port int, deviceID string) error
	GetDeviceIP(ctx context.Context, deviceID string) (string, error)
	RestartServer(ctx context.Context) (string, error)
}

type Device interface {
	DeviceOperator
	DeviceManager
}

func CreateDevice(deviceType string) (Device, error) {
	switch deviceType {
	case constants.ADB:
		return &android.ADBDevice{}, nil
	case constants.IOS:
		return &ios.IOSDevice{}, nil
	default:
		return nil, fmt.Errorf("unknown device type: %v", deviceType)
	}
}
