package audio

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"aprs_agent/config"

	"github.com/gen2brain/malgo"
)

// Input 音频输入
type Input struct {
	config    *config.Config
	devices   DeviceManagerInterface
	device    *malgo.Device
	stream    *malgo.Device
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	level     float64
	gain      float64
	buffer    []byte
	callback  func([]byte, int)
}

// NewInput 创建新的音频输入
func NewInput(cfg *config.Config, devices DeviceManagerInterface) (*Input, error) {
	input := &Input{
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
func (i *Input) Start(ctx context.Context) error {
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

	// 检查设备支持
	if !i.devices.IsDeviceSupported(deviceName, "input", i.config.Audio.Input.SampleRate, i.config.Audio.Input.Channels, i.config.Audio.Input.Format) {
		return fmt.Errorf("设备 %s 不支持指定的配置", deviceName)
	}

	// 创建设备配置
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.SampleRate = uint32(i.config.Audio.Input.SampleRate)
	deviceConfig.PeriodSizeInFrames = uint32(i.config.Audio.Input.BufferSize)
	deviceConfig.Periods = 1
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = uint32(i.config.Audio.Input.Channels)
	// 获取设备上下文
	malgoContext := i.devices.GetContext()
	if malgoContext == nil {
		return fmt.Errorf("macOS设备管理器不支持malgo设备创建，请使用其他音频库")
	}

	// 创建设备
	device, err := malgo.InitDevice(malgoContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: i.dataCallback,
	})
	if err != nil {
		return fmt.Errorf("创建音频输入设备失败: %w", err)
	}

	// 启动设备
	if err := device.Start(); err != nil {
		device.Uninit()
		return fmt.Errorf("启动音频输入设备失败: %w", err)
	}

	i.device = device
	i.isRunning = true
	i.ctx, i.cancel = context.WithCancel(ctx)

	// 启动音频处理协程
	go i.processAudio()

	log.Printf("音频输入已启动: %s", deviceName)
	return nil
}

// Stop 停止音频输入流
func (i *Input) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.isRunning {
		return nil
	}

	if i.cancel != nil {
		i.cancel()
	}

	if i.device != nil {
		i.device.Stop()
		i.device.Uninit()
		i.device = nil
	}

	i.isRunning = false
	log.Println("音频输入已停止")
	return nil
}

// Close 关闭音频输入
func (i *Input) Close() error {
	return i.Stop()
}

// dataCallback 音频数据回调函数
func (i *Input) dataCallback(pOutputSample, pInputSamples []byte, frameCount uint32) {
	if !i.isRunning {
		return
	}

	// 复制音频数据
	copy(i.buffer, pInputSamples)

	// 计算音频级别
	i.calculateLevel(pInputSamples)

	// 应用增益
	if i.gain != 1.0 {
		i.applyGain()
	}

	// 如果有回调函数，调用它
	if i.callback != nil {
		i.callback(i.buffer, int(frameCount))
	}
}

// calculateLevel 计算音频级别
func (i *Input) calculateLevel(data []byte) {
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

// applyGain 应用增益
func (i *Input) applyGain() {
	if i.gain == 1.0 {
		return
	}

	for j := 0; j < len(i.buffer); j += 2 {
		sample := int16(i.buffer[j]) | int16(i.buffer[j+1])<<8
		adjusted := float64(sample) * i.gain

		// 限制在16位范围内
		if adjusted > 32767 {
			adjusted = 32767
		} else if adjusted < -32768 {
			adjusted = -32768
		}

		adjustedSample := int16(adjusted)
		i.buffer[j] = byte(adjustedSample & 0xFF)
		i.buffer[j+1] = byte((adjustedSample >> 8) & 0xFF)
	}
}

// processAudio 音频处理协程
func (i *Input) processAudio() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-ticker.C:
			// 定期处理音频数据
			// 这里可以添加回声消除、噪声抑制等处理
		}
	}
}

// GetLevel 获取当前音频级别
func (i *Input) GetLevel() float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.level
}

// SetGain 设置增益
func (i *Input) SetGain(gain float64) error {
	if gain < 0.0 || gain > 2.0 {
		return fmt.Errorf("增益必须在0.0-2.0之间")
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.gain = gain
	return nil
}

// GetGain 获取当前增益
func (i *Input) GetGain() float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.gain
}

// SetCallback 设置音频数据回调函数
func (i *Input) SetCallback(callback func([]byte, int)) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.callback = callback
}

// IsRunning 检查是否正在运行
func (i *Input) IsRunning() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.isRunning
}

// UpdateConfig 更新配置
func (i *Input) UpdateConfig(newConfig *config.Config) error {
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
func (i *Input) GetBuffer() []byte {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.buffer
}

// GetConfig 获取当前配置
func (i *Input) GetConfig() *config.Config {
	return i.config
}
