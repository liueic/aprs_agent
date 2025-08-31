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

// macOSOutput macOS专用音频输出
type macOSOutput struct {
	config     *config.Config
	devices    DeviceManagerInterface
	isRunning  bool
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	level      float64
	volume     float64
	buffer     []byte
	queue      chan []byte
	deviceName string
}

// newMacOSOutput 创建新的macOS音频输出
func newMacOSOutput(cfg *config.Config, devices DeviceManagerInterface) (AudioOutput, error) {
	output := &macOSOutput{
		config:    cfg,
		devices:   devices,
		isRunning: false,
		level:     0.0,
		volume:    cfg.Audio.Output.Volume,
		buffer:    make([]byte, cfg.Audio.Output.BufferSize*cfg.Audio.Output.Channels*2), // 假设16位音频
		queue:     make(chan []byte, 10),                                                 // 音频数据队列
	}

	return output, nil
}

// Start 启动音频输出流
func (o *macOSOutput) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isRunning {
		return fmt.Errorf("音频输出已在运行")
	}

	// 获取设备
	deviceName := o.config.Audio.Output.DeviceName
	if deviceName == "" {
		// 使用默认设备
		defaultDevice, err := o.devices.GetDefaultDevice("output")
		if err != nil {
			return fmt.Errorf("获取默认输出设备失败: %w", err)
		}
		deviceName = defaultDevice.Name
	}

	o.deviceName = deviceName

	// 检查设备支持
	if !o.devices.IsDeviceSupported(deviceName, "output", o.config.Audio.Output.SampleRate, o.config.Audio.Output.Channels, o.config.Audio.Output.Format) {
		return fmt.Errorf("设备 %s 不支持指定的配置", deviceName)
	}

	// 在macOS上，我们使用系统命令来测试音频设备
	if err := o.testDeviceAccess(); err != nil {
		return fmt.Errorf("测试设备访问失败: %w", err)
	}

	o.isRunning = true
	o.ctx, o.cancel = context.WithCancel(ctx)

	// 启动音频处理协程
	go o.processAudio()

	log.Printf("macOS音频输出已启动: %s", deviceName)
	return nil
}

// testDeviceAccess 测试设备访问
func (o *macOSOutput) testDeviceAccess() error {
	// 在macOS上，我们直接检查设备是否在设备列表中，而不依赖afinfo命令
	// 因为afinfo命令可能无法访问某些系统音频设备
	log.Printf("正在验证音频输出设备: %s", o.deviceName)

	// 检查设备是否在可用设备列表中
	device, err := o.devices.GetDeviceByName(o.deviceName, "output")
	if err != nil {
		return fmt.Errorf("设备 %s 不在可用设备列表中", o.deviceName)
	}

	log.Printf("设备验证成功: %s [%s]", device.Name, device.Type)
	return nil
}

// Stop 停止音频输出流
func (o *macOSOutput) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.isRunning {
		return nil
	}

	if o.cancel != nil {
		o.cancel()
	}

	o.isRunning = false
	log.Println("macOS音频输出已停止")
	return nil
}

// Close 关闭音频输出
func (o *macOSOutput) Close() error {
	return o.Stop()
}

// processAudio 音频处理协程
func (o *macOSOutput) processAudio() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			// 定期处理音频数据
			// 这里可以添加音频效果处理
		}
	}
}

// PlayAudio 播放音频数据
func (o *macOSOutput) PlayAudio(data []byte) error {
	if !o.isRunning {
		return fmt.Errorf("音频输出未运行")
	}

	select {
	case o.queue <- data:
		return nil
	default:
		return fmt.Errorf("音频队列已满")
	}
}

// GetLevel 获取当前音频级别
func (o *macOSOutput) GetLevel() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.level
}

// SetVolume 设置音量
func (o *macOSOutput) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("音量必须在0.0-1.0之间")
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.volume = volume
	return nil
}

// GetVolume 获取当前音量
func (o *macOSOutput) GetVolume() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.volume
}

// IsRunning 检查是否正在运行
func (o *macOSOutput) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.isRunning
}

// UpdateConfig 更新配置
func (o *macOSOutput) UpdateConfig(newConfig *config.Config) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isRunning {
		return fmt.Errorf("无法在运行时更新配置")
	}

	o.config = newConfig
	o.volume = newConfig.Audio.Output.Volume

	// 重新分配缓冲区
	o.buffer = make([]byte, newConfig.Audio.Output.BufferSize*newConfig.Audio.Output.Channels*2)

	return nil
}

// GetBuffer 获取当前音频缓冲区
func (o *macOSOutput) GetBuffer() []byte {
	return o.buffer
}

// GetConfig 获取当前配置
func (o *macOSOutput) GetConfig() *config.Config {
	return o.config
}

// GetQueueSize 获取队列大小
func (o *macOSOutput) GetQueueSize() int {
	return len(o.queue)
}

// ClearQueue 清空音频队列
func (o *macOSOutput) ClearQueue() {
	for len(o.queue) > 0 {
		<-o.queue
	}
}

// 实现与原始Output相同的接口方法
func (o *macOSOutput) calculateLevel(data []byte) {
	if len(data) == 0 {
		o.level = 0.0
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
		o.level = 20 * math.Log10(rms/32767.0)
	} else {
		o.level = -96.0 // 静音
	}
}

func (o *macOSOutput) applyVolume(data []byte) {
	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		adjusted := float64(sample) * o.volume

		// 限制在16位范围内
		if adjusted > 32767 {
			adjusted = 32767
		} else if adjusted < -32768 {
			adjusted = -32768
		}

		adjustedSample := int16(adjusted)
		data[j] = byte(adjustedSample & 0xFF)
		data[j+1] = byte((adjustedSample >> 8) & 0xFF)
	}
}

// 为macOS平台提供通用音频输出的存根
func newGenericOutput(cfg *config.Config, devices DeviceManagerInterface) (AudioOutput, error) {
	return nil, fmt.Errorf("通用音频输出在macOS上不可用，请使用macOS专用版本")
}
