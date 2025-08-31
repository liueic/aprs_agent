//go:build darwin
// +build darwin

package audio

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"aprs_agent/config"
)

// macOSInput macOS专用音频输入
type macOSInput struct {
	config     *config.Config
	devices    DeviceManagerInterface
	isRunning  bool
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	level      float64
	gain       float64
	buffer     []byte
	callback   func([]byte, int)
	deviceName string
}

// newMacOSInput 创建新的macOS音频输入
func newMacOSInput(cfg *config.Config, devices DeviceManagerInterface) (AudioInput, error) {
	input := &macOSInput{
		config:    cfg,
		devices:   devices,
		isRunning: false,
		level:     0.0,
		gain:      cfg.Audio.Input.Gain,
		buffer:    make([]byte, cfg.Audio.Input.BufferSize*cfg.Audio.Input.Channels*2), // 假设16位音频
	}

	return input, nil
}

// Start 启动音频输入流
func (i *macOSInput) Start(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.isRunning {
		return fmt.Errorf("音频输入已在运行")
	}

	// 获取设备
	deviceName := i.config.Audio.Input.DeviceName
	if deviceName == "" {
		// 使用默认设备
		defaultDevice, err := i.devices.GetDefaultDevice("input")
		if err != nil {
			return fmt.Errorf("获取默认输入设备失败: %w", err)
		}
		deviceName = defaultDevice.Name
	}

	i.deviceName = deviceName

	// 检查设备支持
	if !i.devices.IsDeviceSupported(deviceName, "input", i.config.Audio.Input.SampleRate, i.config.Audio.Input.Channels, i.config.Audio.Input.Format) {
		return fmt.Errorf("设备 %s 不支持指定的配置", deviceName)
	}

	// 在macOS上，我们使用系统命令来测试音频设备
	if err := i.testDeviceAccess(); err != nil {
		return fmt.Errorf("测试设备访问失败: %w", err)
	}

	i.isRunning = true
	i.ctx, i.cancel = context.WithCancel(ctx)

	// 启动音频处理协程
	go i.processAudio()

	log.Printf("macOS音频输入已启动: %s", deviceName)
	return nil
}

// testDeviceAccess 测试设备访问
func (i *macOSInput) testDeviceAccess() error {
	// 在macOS上，我们直接检查设备是否在设备列表中，而不依赖afinfo命令
	// 因为afinfo命令可能无法访问某些系统音频设备
	log.Printf("正在验证音频输入设备: %s", i.deviceName)

	// 检查设备是否在可用设备列表中
	device, err := i.devices.GetDeviceByName(i.deviceName, "input")
	if err != nil {
		return fmt.Errorf("设备 %s 不在可用设备列表中", i.deviceName)
	}

	log.Printf("设备验证成功: %s [%s]", device.Name, device.Type)
	return nil
}

// Stop 停止音频输入流
func (i *macOSInput) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.isRunning {
		return nil
	}

	if i.cancel != nil {
		i.cancel()
	}

	i.isRunning = false
	log.Println("macOS音频输入已停止")
	return nil
}

// Close 关闭音频输入
func (i *macOSInput) Close() error {
	return i.Stop()
}

// processAudio 音频处理协程
func (i *macOSInput) processAudio() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-ticker.C:
			// 模拟音频数据（在实际应用中，这里应该从Core Audio获取真实数据）
			i.simulateAudioData()
		}
	}
}

// simulateAudioData 模拟音频数据（用于测试）
func (i *macOSInput) simulateAudioData() {
	// 生成一个简单的正弦波作为测试音频
	sampleRate := i.config.Audio.Input.SampleRate
	channels := i.config.Audio.Input.Channels
	bufferSize := i.config.Audio.Input.BufferSize

	// 计算时间
	now := time.Now()
	timeMs := float64(now.UnixNano()) / 1e9

	// 生成测试音频数据
	for frame := 0; frame < bufferSize; frame++ {
		for ch := 0; ch < channels; ch++ {
			// 生成440Hz的正弦波
			frequency := 440.0
			sample := math.Sin(2 * math.Pi * frequency * (timeMs + float64(frame)/float64(sampleRate)))

			// 应用增益
			sample *= i.gain

			// 转换为16位整数
			sampleInt := int16(sample * 32767)

			// 写入缓冲区
			offset := (frame*channels + ch) * 2
			if offset+1 < len(i.buffer) {
				i.buffer[offset] = byte(sampleInt & 0xFF)
				i.buffer[offset+1] = byte((sampleInt >> 8) & 0xFF)
			}
		}
	}

	// 计算音频级别
	i.calculateLevel(i.buffer)

	// 如果有回调函数，调用它
	if i.callback != nil {
		i.callback(i.buffer, bufferSize)
	}
}

// calculateLevel 计算音频级别
func (i *macOSInput) calculateLevel(data []byte) {
	if len(data) == 0 {
		i.level = 0.0
		return
	}

	// 计算RMS值
	var sum float64
	sampleCount := len(data) / 2 // 假设16位音频

	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		sum += float64(sample * sample)
	}

	rms := math.Sqrt(sum / float64(sampleCount))

	// 转换为分贝
	if rms > 0 {
		i.level = 20 * math.Log10(rms/32767.0)
	} else {
		i.level = -96.0 // 静音
	}
}

// GetLevel 获取当前音频级别
func (i *macOSInput) GetLevel() float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.level
}

// SetGain 设置增益
func (i *macOSInput) SetGain(gain float64) error {
	if gain < 0.0 || gain > 2.0 {
		return fmt.Errorf("增益必须在0.0-2.0之间")
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.gain = gain
	return nil
}

// GetGain 获取当前增益
func (i *macOSInput) GetGain() float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.gain
}

// SetCallback 设置音频数据回调函数
func (i *macOSInput) SetCallback(callback func([]byte, int)) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.callback = callback
}

// IsRunning 检查是否正在运行
func (i *macOSInput) IsRunning() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.isRunning
}

// UpdateConfig 更新配置
func (i *macOSInput) UpdateConfig(newConfig *config.Config) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.isRunning {
		return fmt.Errorf("无法在运行时更新配置")
	}

	i.config = newConfig
	i.gain = newConfig.Audio.Input.Gain

	// 重新分配缓冲区
	i.buffer = make([]byte, newConfig.Audio.Input.BufferSize*newConfig.Audio.Input.Channels*2)

	return nil
}

// GetBuffer 获取当前音频缓冲区
func (i *macOSInput) GetBuffer() []byte {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.buffer
}

// GetConfig 获取当前配置
func (i *macOSInput) GetConfig() *config.Config {
	return i.config
}

// 为macOS平台提供通用音频输入输出的存根
func newGenericInput(cfg *config.Config, devices DeviceManagerInterface) (AudioInput, error) {
	return nil, fmt.Errorf("通用音频输入在macOS上不可用，请使用macOS专用版本")
}
