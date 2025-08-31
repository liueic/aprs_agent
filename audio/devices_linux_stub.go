//go:build !linux

package audio

import "fmt"

// newLinuxDeviceManager 创建Linux设备管理器（非Linux系统存根）
func newLinuxDeviceManager() (DeviceManagerInterface, error) {
	return nil, fmt.Errorf("Linux设备管理器仅在Linux系统上可用")
}
