package audio

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/gen2brain/malgo"
)

// DeviceInfo 音频设备信息
type DeviceInfo struct {
	ID          string
	Name        string
	Type        string
	SampleRates []int
	Channels    []int
	Formats     []string
	IsDefault   bool
}

// DeviceManagerInterface 音频设备管理器接口
type DeviceManagerInterface interface {
	ListDevices()
	GetDeviceByName(name string, deviceType string) (*DeviceInfo, error)
	GetDefaultDevice(deviceType string) (*DeviceInfo, error)
	GetDevicesByType(deviceType string) []DeviceInfo
	GetAllDevices() []DeviceInfo
	GetDeviceCount() int
	RefreshDevices() error
	Close() error
	IsDeviceSupported(deviceName string, deviceType string, sampleRate int, channels int, format string) bool
	GetContext() *malgo.AllocatedContext
}

// DeviceManager 音频设备管理器
type DeviceManager struct {
	context *malgo.AllocatedContext
	devices []DeviceInfo
}

// NewDeviceManager 创建新的设备管理器
func NewDeviceManager() (DeviceManagerInterface, error) {
	// 在macOS上使用专用管理器
	if runtime.GOOS == "darwin" {
		return newMacOSDeviceManager()
	}

	// 在Linux上使用专用管理器
	if runtime.GOOS == "linux" {
		return newLinuxDeviceManager()
	}

	// 其他系统使用malgo
	context, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("初始化音频上下文失败: %w", err)
	}

	manager := &DeviceManager{
		context: context,
		devices: []DeviceInfo{},
	}

	// 枚举设备
	if err := manager.enumerateDevices(); err != nil {
		context.Uninit()
		return nil, fmt.Errorf("枚举音频设备失败: %w", err)
	}

	return manager, nil
}

// enumerateDevices 枚举音频设备
func (dm *DeviceManager) enumerateDevices() error {
	// 枚举输入设备
	inputDevices, err := dm.context.Devices(malgo.Capture)
	if err != nil {
		return fmt.Errorf("枚举输入设备失败: %w", err)
	}

	// 枚举输出设备
	outputDevices, err := dm.context.Devices(malgo.Playback)
	if err != nil {
		return fmt.Errorf("枚举输出设备失败: %w", err)
	}

	// 处理输入设备
	for _, device := range inputDevices {
		deviceInfo := DeviceInfo{
			ID:          device.ID.String(),
			Name:        strings.TrimSpace(device.Name()),
			Type:        "input",
			SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
			Channels:    []int{1, 2, 4, 6, 8},
			Formats:     []string{"int16", "float32"},
			IsDefault:   device.IsDefault != 0,
		}
		dm.devices = append(dm.devices, deviceInfo)
	}

	// 处理输出设备
	for _, device := range outputDevices {
		deviceInfo := DeviceInfo{
			ID:          device.ID.String(),
			Name:        strings.TrimSpace(device.Name()),
			Type:        "output",
			SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
			Channels:    []int{1, 2, 4, 6, 8},
			Formats:     []string{"int16", "float32"},
			IsDefault:   device.IsDefault != 0,
		}
		dm.devices = append(dm.devices, deviceInfo)
	}

	return nil
}

// ListDevices 列出所有音频设备
func (dm *DeviceManager) ListDevices() {
	log.Println("=== 可用的音频设备 ===")

	if len(dm.devices) == 0 {
		log.Println("未找到音频设备")
		return
	}

	for i, device := range dm.devices {
		defaultMark := ""
		if device.IsDefault {
			defaultMark = " (默认)"
		}

		log.Printf("%d. %s%s [%s]", i+1, device.Name, defaultMark, device.Type)
		log.Printf("   ID: %s", device.ID)
		log.Printf("   支持的采样率: %v", device.SampleRates)
		log.Printf("   支持的声道数: %v", device.Channels)
		log.Printf("   支持的格式: %v", device.Formats)
		log.Println()
	}
}

// GetDeviceByName 根据名称获取设备
func (dm *DeviceManager) GetDeviceByName(name string, deviceType string) (*DeviceInfo, error) {
	for _, device := range dm.devices {
		if device.Name == name && device.Type == deviceType {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到设备: %s [%s]", name, deviceType)
}

// GetDefaultDevice 获取默认设备
func (dm *DeviceManager) GetDefaultDevice(deviceType string) (*DeviceInfo, error) {
	for _, device := range dm.devices {
		if device.Type == deviceType && device.IsDefault {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到默认的 %s 设备", deviceType)
}

// GetDevicesByType 根据类型获取设备列表
func (dm *DeviceManager) GetDevicesByType(deviceType string) []DeviceInfo {
	var result []DeviceInfo
	for _, device := range dm.devices {
		if device.Type == deviceType {
			result = append(result, device)
		}
	}
	return result
}

// GetAllDevices 获取所有设备
func (dm *DeviceManager) GetAllDevices() []DeviceInfo {
	return dm.devices
}

// GetDeviceCount 获取设备数量
func (dm *DeviceManager) GetDeviceCount() int {
	return len(dm.devices)
}

// RefreshDevices 刷新设备列表
func (dm *DeviceManager) RefreshDevices() error {
	dm.devices = []DeviceInfo{}
	return dm.enumerateDevices()
}

// Close 关闭设备管理器
func (dm *DeviceManager) Close() error {
	if dm.context != nil {
		dm.context.Uninit()
	}
	return nil
}

// GetContext 获取音频上下文
func (dm *DeviceManager) GetContext() *malgo.AllocatedContext {
	return dm.context
}

// IsDeviceSupported 检查设备是否支持指定的配置
func (dm *DeviceManager) IsDeviceSupported(deviceName string, deviceType string, sampleRate int, channels int, format string) bool {
	device, err := dm.GetDeviceByName(deviceName, deviceType)
	if err != nil {
		return false
	}

	// 检查采样率
	sampleRateSupported := false
	for _, sr := range device.SampleRates {
		if sr == sampleRate {
			sampleRateSupported = true
			break
		}
	}

	// 检查声道数
	channelsSupported := false
	for _, ch := range device.Channels {
		if ch == channels {
			channelsSupported = true
			break
		}
	}

	// 检查格式
	formatSupported := false
	for _, fmt := range device.Formats {
		if fmt == format {
			formatSupported = true
			break
		}
	}

	return sampleRateSupported && channelsSupported && formatSupported
}
