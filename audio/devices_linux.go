//go:build linux

package audio

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// LinuxDeviceManager Linux专用音频设备管理器
type LinuxDeviceManager struct {
	devices []DeviceInfo
	context interface{} // 使用interface{}避免编译时类型问题
}

// newLinuxDeviceManager 创建新的Linux设备管理器
func newLinuxDeviceManager() (DeviceManagerInterface, error) {
	manager := &LinuxDeviceManager{
		devices: []DeviceInfo{},
	}

	// 尝试使用系统命令枚举设备
	if err := manager.enumerateDevicesWithCommands(); err != nil {
		log.Printf("系统命令枚举失败，回退到malgo: %v", err)
		// 回退到malgo
		if err := manager.enumerateDevicesWithMalgo(); err != nil {
			return nil, fmt.Errorf("所有设备枚举方法都失败: %w", err)
		}
	}

	return manager, nil
}

// enumerateDevicesWithCommands 使用系统命令枚举音频设备
func (dm *LinuxDeviceManager) enumerateDevicesWithCommands() error {
	// 尝试使用pactl命令（PulseAudio）
	if dm.tryPulseAudioDevices() {
		return nil
	}

	// 尝试使用amixer命令（ALSA）
	if dm.tryALSADevices() {
		return nil
	}

	// 尝试使用aplay/arecord命令
	if dm.tryALSACommands() {
		return nil
	}

	return fmt.Errorf("所有系统命令都失败")
}

// tryPulseAudioDevices 尝试使用PulseAudio枚举设备
func (dm *LinuxDeviceManager) tryPulseAudioDevices() bool {
	// 检查pactl是否可用
	if _, err := exec.LookPath("pactl"); err != nil {
		return false
	}

	log.Println("使用PulseAudio枚举音频设备...")

	// 获取输入设备
	cmd := exec.Command("pactl", "list", "short", "sources")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("pactl sources失败: %v", err)
		return false
	}

	// 解析输入设备
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deviceName := strings.TrimSpace(parts[1])
			// 跳过监控设备
			if !strings.Contains(strings.ToLower(deviceName), "monitor") {
				dm.devices = append(dm.devices, DeviceInfo{
					ID:          parts[0],
					Name:        deviceName,
					Type:        "input",
					SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
					Channels:    []int{1, 2},
					Formats:     []string{"int16", "float32"},
					IsDefault:   false, // 稍后检查
				})
			}
		}
	}

	// 获取输出设备
	cmd = exec.Command("pactl", "list", "short", "sinks")
	output, err = cmd.Output()
	if err != nil {
		log.Printf("pactl sinks失败: %v", err)
		return false
	}

	// 解析输出设备
	lines = strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deviceName := strings.TrimSpace(parts[1])
			dm.devices = append(dm.devices, DeviceInfo{
				ID:          parts[0],
				Name:        deviceName,
				Type:        "output",
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
				Channels:    []int{1, 2},
				Formats:     []string{"int16", "float32"},
				IsDefault:   false, // 稍后检查
			})
		}
	}

	// 检查默认设备
	dm.checkPulseAudioDefaults()

	return len(dm.devices) > 0
}

// checkPulseAudioDefaults 检查PulseAudio默认设备
func (dm *LinuxDeviceManager) checkPulseAudioDefaults() {
	// 获取默认输入设备
	cmd := exec.Command("pactl", "get-default-source")
	if output, err := cmd.Output(); err == nil {
		defaultInput := strings.TrimSpace(string(output))
		for i := range dm.devices {
			if dm.devices[i].Type == "input" && strings.Contains(dm.devices[i].Name, defaultInput) {
				dm.devices[i].IsDefault = true
				break
			}
		}
	}

	// 获取默认输出设备
	cmd = exec.Command("pactl", "get-default-sink")
	if output, err := cmd.Output(); err == nil {
		defaultOutput := strings.TrimSpace(string(output))
		for i := range dm.devices {
			if dm.devices[i].Type == "output" && strings.Contains(dm.devices[i].Name, defaultOutput) {
				dm.devices[i].IsDefault = true
				break
			}
		}
	}
}

// tryALSADevices 尝试使用ALSA枚举设备
func (dm *LinuxDeviceManager) tryALSADevices() bool {
	// 检查amixer是否可用
	if _, err := exec.LookPath("amixer"); err != nil {
		return false
	}

	log.Println("使用ALSA枚举音频设备...")

	// 获取控制设备列表
	cmd := exec.Command("amixer", "scontrols")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 解析控制设备
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Simple mixer control") {
			continue
		}
		deviceName := strings.TrimSpace(line)
		if deviceName != "" {
			// 根据名称推断类型
			deviceType := "input"
			if strings.Contains(strings.ToLower(deviceName), "master") ||
				strings.Contains(strings.ToLower(deviceName), "pcm") ||
				strings.Contains(strings.ToLower(deviceName), "speaker") {
				deviceType = "output"
			}

			dm.devices = append(dm.devices, DeviceInfo{
				ID:          deviceName,
				Name:        deviceName,
				Type:        deviceType,
				SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
				Channels:    []int{1, 2},
				Formats:     []string{"int16", "float32"},
				IsDefault:   false,
			})
		}
	}

	return len(dm.devices) > 0
}

// tryALSACommands 尝试使用aplay/arecord命令
func (dm *LinuxDeviceManager) tryALSACommands() bool {
	log.Println("使用ALSA命令枚举音频设备...")

	// 尝试aplay -l
	if dm.tryAplayDevices() {
		return true
	}

	// 尝试arecord -l
	if dm.tryArecordDevices() {
		return true
	}

	return false
}

// tryAplayDevices 尝试使用aplay命令
func (dm *LinuxDeviceManager) tryAplayDevices() bool {
	cmd := exec.Command("aplay", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "card") && strings.Contains(line, "device") {
			// 提取设备名称
			re := regexp.MustCompile(`card \d+: (.+?) \[(.+?)\]`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				deviceName := strings.TrimSpace(matches[1])

				dm.devices = append(dm.devices, DeviceInfo{
					ID:          deviceName,
					Name:        deviceName,
					Type:        "output", // aplay是播放设备
					SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
					Channels:    []int{1, 2},
					Formats:     []string{"int16", "float32"},
					IsDefault:   false,
				})
			}
		}
	}

	return len(dm.devices) > 0
}

// tryArecordDevices 尝试使用arecord命令
func (dm *LinuxDeviceManager) tryArecordDevices() bool {
	cmd := exec.Command("arecord", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "card") && strings.Contains(line, "device") {
			// 提取设备名称
			re := regexp.MustCompile(`card \d+: (.+?) \[(.+?)\]`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				deviceName := strings.TrimSpace(matches[1])

				dm.devices = append(dm.devices, DeviceInfo{
					ID:          deviceName,
					Name:        deviceName,
					Type:        "input", // arecord是录音设备
					SampleRates: []int{8000, 11025, 16000, 22050, 44100, 48000, 96000},
					Channels:    []int{1, 2},
					Formats:     []string{"int16", "float32"},
					IsDefault:   false,
				})
			}
		}
	}

	return len(dm.devices) > 0
}

// enumerateDevicesWithMalgo 使用malgo枚举设备（回退方法）
func (dm *LinuxDeviceManager) enumerateDevicesWithMalgo() error {
	// 简化版本：直接返回错误，让系统命令处理
	return fmt.Errorf("malgo回退方法暂不可用")
}

// ListDevices 列出所有音频设备
func (dm *LinuxDeviceManager) ListDevices() {
	log.Println("=== Linux音频设备 ===")

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
func (dm *LinuxDeviceManager) GetDeviceByName(name string, deviceType string) (*DeviceInfo, error) {
	for _, device := range dm.devices {
		if device.Name == name && device.Type == deviceType {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("未找到设备: %s [%s]", name, deviceType)
}

// GetDefaultDevice 获取默认设备
func (dm *LinuxDeviceManager) GetDefaultDevice(deviceType string) (*DeviceInfo, error) {
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
func (dm *LinuxDeviceManager) GetDevicesByType(deviceType string) []DeviceInfo {
	var result []DeviceInfo
	for _, device := range dm.devices {
		if device.Type == deviceType {
			result = append(result, device)
		}
	}
	return result
}

// GetAllDevices 获取所有设备
func (dm *LinuxDeviceManager) GetAllDevices() []DeviceInfo {
	return dm.devices
}

// GetDeviceCount 获取设备数量
func (dm *LinuxDeviceManager) GetDeviceCount() int {
	return len(dm.devices)
}

// RefreshDevices 刷新设备列表
func (dm *LinuxDeviceManager) RefreshDevices() error {
	dm.devices = []DeviceInfo{}
	return dm.enumerateDevicesWithCommands()
}

// Close 关闭设备管理器
func (dm *LinuxDeviceManager) Close() error {
	// Linux设备管理器不需要特殊清理
	return nil
}

// GetContext 获取音频上下文
func (dm *LinuxDeviceManager) GetContext() interface{} {
	return dm.context
}

// IsDeviceSupported 检查设备是否支持指定的配置
func (dm *LinuxDeviceManager) IsDeviceSupported(deviceName string, deviceType string, sampleRate int, channels int, format string) bool {
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
