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

// Output 音频输出
type Output struct {
	config    *config.Config
	devices   DeviceManagerInterface
	device    *malgo.Device
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	level     float64
	volume    float64
	buffer    []byte
	queue     chan []byte
}

// NewOutput 创建新的音频输出
func NewOutput(cfg *config.Config, devices DeviceManagerInterface) (*Output, error) {
	output := &Output{
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
func (o *Output) Start(ctx context.Context) error {
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

	// 检查设备支持
	if !o.devices.IsDeviceSupported(deviceName, "output", o.config.Audio.Output.SampleRate, o.config.Audio.Output.Channels, o.config.Audio.Output.Format) {
		return fmt.Errorf("设备 %s 不支持指定的配置", deviceName)
	}

	// 创建设备配置
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.SampleRate = uint32(o.config.Audio.Output.SampleRate)
	deviceConfig.PeriodSizeInFrames = uint32(o.config.Audio.Output.BufferSize)
	deviceConfig.Periods = 1
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = uint32(o.config.Audio.Output.Channels)

	// 获取设备上下文
	malgoContext := o.devices.GetContext()
	if malgoContext == nil {
		return fmt.Errorf("macOS设备管理器不支持malgo设备创建，请使用其他音频库")
	}

	// 创建设备
	device, err := malgo.InitDevice(malgoContext.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: o.dataCallback,
	})
	if err != nil {
		return fmt.Errorf("创建音频输出设备失败: %w", err)
	}

	// 启动设备
	if err := device.Start(); err != nil {
		device.Uninit()
		return fmt.Errorf("启动音频输出设备失败: %w", err)
	}

	o.device = device
	o.isRunning = true
	o.ctx, o.cancel = context.WithCancel(ctx)

	// 启动音频处理协程
	go o.processAudio()

	log.Printf("音频输出已启动: %s", deviceName)
	return nil
}

// Stop 停止音频输出流
func (o *Output) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.isRunning {
		return nil
	}

	if o.cancel != nil {
		o.cancel()
	}

	if o.device != nil {
		o.device.Stop()
		o.device.Uninit()
		o.device = nil
	}

	o.isRunning = false
	log.Println("音频输出已停止")
	return nil
}

// Close 关闭音频输出
func (o *Output) Close() error {
	return o.Stop()
}

// dataCallback 音频数据回调函数
func (o *Output) dataCallback(pOutputSample, pInputSamples []byte, frameCount uint32) {
	if !o.isRunning {
		return
	}

	// 从队列获取音频数据
	select {
	case data := <-o.queue:
		// 应用音量
		if o.volume != 1.0 {
			o.applyVolume(data)
		}

		// 复制到输出缓冲区
		copy(pOutputSample, data)

		// 计算音频级别
		o.calculateLevel(data)

	default:
		// 队列为空，输出静音
		for i := range pOutputSample {
			pOutputSample[i] = 0
		}
		o.level = -96.0
	}
}

// applyVolume 应用音量
func (o *Output) applyVolume(data []byte) {
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

// calculateLevel 计算音频级别
func (o *Output) calculateLevel(data []byte) {
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

// processAudio 音频处理协程
func (o *Output) processAudio() {
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
func (o *Output) PlayAudio(data []byte) error {
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
func (o *Output) GetLevel() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.level
}

// SetVolume 设置音量
func (o *Output) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("音量必须在0.0-1.0之间")
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.volume = volume
	return nil
}

// GetVolume 获取当前音量
func (o *Output) GetVolume() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.volume
}

// IsRunning 检查是否正在运行
func (o *Output) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.isRunning
}

// UpdateConfig 更新配置
func (o *Output) UpdateConfig(newConfig *config.Config) error {
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
func (o *Output) GetBuffer() []byte {
	return o.buffer
}

// GetConfig 获取当前配置
func (o *Output) GetConfig() *config.Config {
	return o.config
}

// GetQueueSize 获取队列大小
func (o *Output) GetQueueSize() int {
	return len(o.queue)
}

// ClearQueue 清空音频队列
func (o *Output) ClearQueue() {
	for len(o.queue) > 0 {
		<-o.queue
	}
}
