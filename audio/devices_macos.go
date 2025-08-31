//go:build darwin

package audio

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/gen2brain/malgo"
)

// macOSDeviceManager 使用Core Audio的macOS专用设备管理器
type macOSDeviceManager struct {
	devices []DeviceInfo
}

// newMacOSDeviceManager 创建新的macOS设备管理器
func newMacOSDeviceManager() (DeviceManagerInterface, error) {
	manager := &macOSDeviceManager{
		devices: []DeviceInfo{},
	}

	if err := manager.enumerateDevices(); err != nil {
		return nil, fmt.Errorf("枚举macOS音频设备失败: %w", err)
	}

	return manager, nil
}

// enumerateDevices 枚举macOS音频设备
func (dm *macOSDeviceManager) enumerateDevices() error {
	// 使用system_profiler命令获取音频设备信息
	cmd := exec.Command("system_profiler", "SPAudioDataType")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("执行system_profiler失败: %w", err)
	}

	outputStr := string(output)

	// 解析输出，提取设备信息
	dm.parseSystemProfilerOutput(outputStr)

	// 如果没有找到设备，尝试使用其他方法
	if len(dm.devices) == 0 {
		dm.fallbackDeviceEnumeration()
	}

	return nil
}

// parseSystemProfilerOutput 解析system_profiler的输出
func (dm *macOSDeviceManager) parseSystemProfilerOutput(output string) {
	lines := strings.Split(output, "\n")

	var currentDevice *DeviceInfo
	var inDeviceSection bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查是否进入设备部分
		if strings.Contains(line, "Devices:") {
			inDeviceSection = true
			continue
		}

		if !inDeviceSection {
			continue
		}

		// 检查设备名称（以冒号结尾的行通常是设备名）
		if strings.HasSuffix(line, ":") && !strings.Contains(line, "Channels:") &&
			!strings.Contains(line, "Manufacturer:") && !strings.Contains(line, "SampleRate:") {
			deviceName := strings.TrimSuffix(line, ":")

			// 跳过一些系统信息行
			if deviceName == "Audio" || deviceName == "Devices" {
				continue
			}

			// 创建新设备
			if currentDevice != nil {
				dm.devices = append(dm.devices, *currentDevice)
			}

			currentDevice = &DeviceInfo{
				Name:        deviceName,
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
				Channels:    []int{1, 2},
				Formats:     []string{"int16", "float32"},
				IsDefault:   false,
			}
		}

		// 检查是否为默认设备
		if strings.Contains(line, "Default Input Device: Yes") {
			if currentDevice != nil {
				currentDevice.Type = "input"
				currentDevice.IsDefault = true
			}
		} else if strings.Contains(line, "Default Output Device: Yes") {
			if currentDevice != nil {
				currentDevice.Type = "output"
				currentDevice.IsDefault = true
			}
		}

		// 检查输入/输出声道数
		if strings.Contains(line, "Input Channels:") {
			if currentDevice != nil && currentDevice.Type == "" {
				currentDevice.Type = "input"
			}
		} else if strings.Contains(line, "Output Channels:") {
			if currentDevice != nil && currentDevice.Type == "" {
				currentDevice.Type = "output"
			}
		}

		// 检查是否为系统输出设备
		if strings.Contains(line, "Default System Output Device: Yes") {
			if currentDevice != nil && currentDevice.Type == "" {
				currentDevice.Type = "output"
			}
		}
	}

	// 添加最后一个设备
	if currentDevice != nil {
		dm.devices = append(dm.devices, *currentDevice)
	}

	// 如果没有找到任何设备，使用备用方法
	if len(dm.devices) == 0 {
		log.Println("system_profiler未返回设备信息，使用备用方法...")
		dm.fallbackDeviceEnumeration()
		return
	}

	// 确保所有设备都有类型
	for i := range dm.devices {
		if dm.devices[i].Type == "" {
			// 根据设备名称推断类型
			if strings.Contains(strings.ToLower(dm.devices[i].Name), "麦克风") ||
				strings.Contains(strings.ToLower(dm.devices[i].Name), "microphone") ||
				strings.Contains(strings.ToLower(dm.devices[i].Name), "input") {
				dm.devices[i].Type = "input"
			} else if strings.Contains(strings.ToLower(dm.devices[i].Name), "扬声器") ||
				strings.Contains(strings.ToLower(dm.devices[i].Name), "speaker") ||
				strings.Contains(strings.ToLower(dm.devices[i].Name), "output") {
				dm.devices[i].Type = "output"
			} else {
				// 默认为输入设备
				dm.devices[i].Type = "input"
			}
		}
	}

	log.Printf("成功解析到 %d 个音频设备", len(dm.devices))
}

// fallbackDeviceEnumeration 备用设备枚举方法
func (dm *macOSDeviceManager) fallbackDeviceEnumeration() {
	log.Println("使用备用方法枚举音频设备...")

	// 尝试使用SwitchAudioSource命令（如果安装了的话）
	if dm.trySwitchAudioSource() {
		return
	}

	// 最后的备用方案：创建默认设备
	dm.createDefaultDevices()
}

// trySwitchAudioSource 尝试使用SwitchAudioSource命令
func (dm *macOSDeviceManager) trySwitchAudioSource() bool {
	// 检查是否安装了SwitchAudioSource
	if _, err := exec.LookPath("SwitchAudioSource"); err != nil {
		return false
	}

	// 获取输入设备
	cmd := exec.Command("SwitchAudioSource", "-a", "-t", "input")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	inputDevices := strings.Split(strings.TrimSpace(string(output)), "\n")

	// 获取输出设备
	cmd = exec.Command("SwitchAudioSource", "-a", "-t", "output")
	output, err = cmd.Output()
	if err != nil {
		return false
	}

	outputDevices := strings.Split(strings.TrimSpace(string(output)), "\n")

	// 处理输入设备
	for _, device := range inputDevices {
		if device != "" {
			dm.devices = append(dm.devices, DeviceInfo{
				Name:        device,
				Type:        "input",
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
				Channels:    []int{1, 2},
				Formats:     []string{"int16", "float32"},
				IsDefault:   false,
			})
		}
	}

	// 处理输出设备
	for _, device := range outputDevices {
		if device != "" {
			dm.devices = append(dm.devices, DeviceInfo{
				Name:        device,
				Type:        "output",
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
				Channels:    []int{1, 2},
				Formats:     []string{"int16", "float32"},
				IsDefault:   false,
			})
		}
	}

	return len(dm.devices) > 0
}

// createDefaultDevices 创建默认设备
func (dm *macOSDeviceManager) createDefaultDevices() {
	log.Println("创建默认音频设备...")

	// 创建默认输入设备
	dm.devices = append(dm.devices, DeviceInfo{
		Name:        "MacBook Air麦克风",
		Type:        "input",
		SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
		Channels:    []int{1, 2},
		Formats:     []string{"int16", "float32"},
		IsDefault:   true,
	})

	// 创建默认输出设备
	dm.devices = append(dm.devices, DeviceInfo{
		Name:        "MacBook Air扬声器",
		Type:        "output",
		SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
		Channels:    []int{1, 2},
		Formats:     []string{"int16", "float32"},
		IsDefault:   true,
	})
}

// ListDevices 列出所有音频设备
func (dm *macOSDeviceManager) ListDevices() {
	log.Println("=== macOS音频设备 ===")

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
		log.Printf("   支持的采样率: %v", device.SampleRates)
		log.Printf("   支持的声道数: %v", device.Channels)
		log.Printf("   支持的格式: %v", device.Formats)
		log.Println()
	}
}

// GetDeviceByName 根据名称获取设备
func (dm *macOSDeviceManager) GetDeviceByName(name string, deviceType string) (*DeviceInfo, error) {
	for _, device := range dm.devices {
		if device.Name == name && device.Type == deviceType {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到设备: %s [%s]", name, deviceType)
}

// GetDefaultDevice 获取默认设备
func (dm *macOSDeviceManager) GetDefaultDevice(deviceType string) (*DeviceInfo, error) {
	for _, device := range dm.devices {
		if device.Type == deviceType && device.IsDefault {
			return &device, nil
		}
	}

	// 如果没有找到默认设备，返回第一个匹配类型的设备
	for _, device := range dm.devices {
		if device.Type == deviceType {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("未找到 %s 设备", deviceType)
}

// GetDevicesByType 根据类型获取设备列表
func (dm *macOSDeviceManager) GetDevicesByType(deviceType string) []DeviceInfo {
	var result []DeviceInfo
	for _, device := range dm.devices {
		if device.Type == deviceType {
			result = append(result, device)
		}
	}
	return result
}

// GetAllDevices 获取所有设备
func (dm *macOSDeviceManager) GetAllDevices() []DeviceInfo {
	return dm.devices
}

// GetDeviceCount 获取设备数量
func (dm *macOSDeviceManager) GetDeviceCount() int {
	return len(dm.devices)
}

// RefreshDevices 刷新设备列表
func (dm *macOSDeviceManager) RefreshDevices() error {
	dm.devices = []DeviceInfo{}
	return dm.enumerateDevices()
}

// Close 关闭设备管理器
func (dm *macOSDeviceManager) Close() error {
	// macOS设备管理器不需要特殊清理
	return nil
}

// GetContext 获取音频上下文（macOS版本返回nil）
func (dm *macOSDeviceManager) GetContext() *malgo.AllocatedContext {
	return nil
}

// IsDeviceSupported 检查设备是否支持指定的配置
func (dm *macOSDeviceManager) IsDeviceSupported(deviceName string, deviceType string, sampleRate int, channels int, format string) bool {
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
