package android

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"autoglm-go/phoneagent/definitions"
	logs "github.com/sirupsen/logrus"
)

const (
	adbPath = "adb"
)

func (r *ADBDevice) Connect(ctx context.Context, address string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Prepare the command
	cmdArgs := []string{"connect", address}

	logs.Debugf("[Connect] run cmd: %s %s", adbPath, strings.Join(cmdArgs, " "))

	cmd := exec.CommandContext(ctx, adbPath, cmdArgs...)
	rawOutput, err := cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[Connect] run cmd failed, err: %v", err)
		return fmt.Sprintf("Connect error: %v", err), err
	}

	logs.Debugf("[Connect] raw output: %s", rawOutput)

	output := string(rawOutput)

	lowerOutput := strings.ToLower(output)

	if strings.Contains(lowerOutput, "already connected") {
		return fmt.Sprintf("Already connected to %s", address), nil
	}
	if strings.Contains(lowerOutput, " connected") {
		return fmt.Sprintf("Connected to %s", address), nil
	}

	return fmt.Sprintf("Connection error: %s", strings.TrimSpace(output)), nil
}

func (r *ADBDevice) Disconnect(ctx context.Context, address string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmdArgs := []string{"disconnect"}
	if len(address) > 0 {
		cmdArgs = append(cmdArgs, address)
	}
	logs.Debugf("[Disconnect] run cmd: %s %s", adbPath, strings.Join(cmdArgs, " "))

	cmd := exec.CommandContext(ctx, adbPath, cmdArgs...)
	rawOutput, err := cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[Disconnect] run cmd failed, err: %v", err)
		return fmt.Sprintf("Disconnect error: %v", err), err
	}

	logs.Debugf("[Disconnect] raw output: %s", rawOutput)

	return string(rawOutput), nil
}

func (r *ADBDevice) ListDevices(ctx context.Context) ([]definitions.DeviceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmdArgs := []string{"devices", "-l"}

	logs.Debugf("[ListDevices] run cmd: %s %s", adbPath, strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, adbPath, cmdArgs...)

	rawOutput, err := cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[ListDevices] run cmd failed, err: %v", err)
		return nil, err
	}
	output := string(rawOutput)

	var devices []definitions.DeviceInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Skip the first line (header)
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		deviceID := parts[0]
		status := parts[1]

		// Determine connection type
		var connType definitions.ConnectionType
		if strings.Contains(deviceID, ":") {
			connType = definitions.Remote
		} else if strings.Contains(deviceID, "emulator") {
			connType = definitions.USB // Emulator via USB
		} else {
			connType = definitions.USB
		}

		// Parse additional info
		var model string
		for _, part := range parts[2:] {
			if strings.HasPrefix(part, "model:") {
				model = strings.SplitN(part, ":", 2)[1]
				break
			}
		}

		devices = append(devices, definitions.DeviceInfo{
			DeviceID:       deviceID,
			Status:         status,
			ConnectionType: connType,
			Model:          model,
		})
	}

	return devices, nil
}

func (r *ADBDevice) GetDeviceInfo(ctx context.Context, deviceID string) (*definitions.DeviceInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (r *ADBDevice) IsConnected(ctx context.Context, deviceID string) bool {
	// TODO implement me
	panic("implement me")
}

func (r *ADBDevice) EnableTCPIP(ctx context.Context, port int, deviceID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Prepare the command
	var cmdArgs []string
	if len(deviceID) > 0 {
		cmdArgs = append(cmdArgs, "-s", deviceID)
	}
	cmdArgs = append(cmdArgs, "tcpip", strconv.Itoa(port))

	logs.Debugf("[EnableTCPIP] run cmd: %s %s", adbPath, strings.Join(cmdArgs, " "))

	cmd := exec.CommandContext(ctx, adbPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[EnableTCPIP] run cmd failed, err: %v", err)
		return err
	}

	// Process results
	if strings.Contains(strings.ToLower(string(output)), "restarting") {
		time.Sleep(time.Second * 5)
		return nil
	}
	return fmt.Errorf("error enabling TCP/IP: %v", err)
}

func (r *ADBDevice) GetDeviceIP(ctx context.Context, deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ---------- 1. adb shell ip route ----------
	var cmdArgs []string
	if len(deviceID) > 0 {
		cmdArgs = append(cmdArgs, "-s", deviceID)
	}
	cmdArgs = append(cmdArgs, "shell", "ip", "route")

	cmd := exec.CommandContext(ctx, adbPath, cmdArgs...)
	logs.Debugf("[GetDeviceIP] run cmd1: %s %s\n", adbPath, strings.Join(cmdArgs, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[GetDeviceIP] run cmd1 failed, err: %v\n", err)
		return "", err
	}

	logs.Debugf("[GetDeviceIP] output1: %s\n", string(output))

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "src") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "src" && i+1 < len(parts) {
					ip := parts[i+1]
					return ip, nil
				}
			}
		}
	}

	// ---------- 2. adb shell ip addr show wlan0 ----------
	cmdArgs = []string{}
	if len(deviceID) > 0 {
		cmdArgs = append(cmdArgs, "-s", deviceID)
	}
	cmdArgs = append(cmdArgs, "shell", "ip", "addr", "show", "wlan0")
	logs.Debugf("[GetDeviceIP] run cmd2: %s %s\n", adbPath, strings.Join(cmdArgs, " "))

	cmd = exec.CommandContext(ctx, adbPath, cmdArgs...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		logs.Errorf("[GetDeviceIP] run cmd2 failed, err: %v\n", err)
		return "", err
	}
	logs.Debugf("[GetDeviceIP] output2: %s\n", string(output))

	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.Split(parts[1], "/")[0]
				return ip, nil
			}
		}
	}
	return "", nil
}

func (r *ADBDevice) RestartServer(ctx context.Context) (string, error) {
	// TODO implement me
	panic("implement me")
}
