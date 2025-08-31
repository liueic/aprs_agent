package audio

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"

	"aprs_agent/config"
)

// AudioInput 音频输入接口
type AudioInput interface {
	Start(ctx context.Context) error
	Stop() error
	Close() error
	GetLevel() float64
	SetGain(gain float64) error
	GetGain() float64
	SetCallback(callback func([]byte, int))
	IsRunning() bool
	UpdateConfig(newConfig *config.Config) error
	GetBuffer() []byte
	GetConfig() *config.Config
}

// AudioOutput 音频输出接口
type AudioOutput interface {
	Start(ctx context.Context) error
	Stop() error
	Close() error
	PlayAudio(data []byte) error
	GetLevel() float64
	SetVolume(volume float64) error
	GetVolume() float64
	IsRunning() bool
	UpdateConfig(newConfig *config.Config) error
	GetBuffer() []byte
	GetConfig() *config.Config
	GetQueueSize() int
	ClearQueue()
}

// Manager 音频管理器
type Manager struct {
	config        *config.Config
	input         AudioInput
	output        AudioOutput
	devices       DeviceManagerInterface
	aprsProcessor *APRSProcessor
	mu            sync.RWMutex
	isRunning     bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager 创建新的音频管理器
func NewManager(cfg *config.Config) (*Manager, error) {
	devices, err := NewDeviceManager()
	if err != nil {
		return nil, fmt.Errorf("创建设备管理器失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:        cfg,
		devices:       devices,
		aprsProcessor: NewAPRSProcessor(),
		ctx:           ctx,
		cancel:        cancel,
		isRunning:     false,
	}

	// 创建音频输入
	if runtime.GOOS == "darwin" {
		input, err := NewmacOSInput(cfg, devices)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建macOS音频输入失败: %w", err)
		}
		manager.input = input

		output, err := NewmacOSOutput(cfg, devices)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建macOS音频输出失败: %w", err)
		}
		manager.output = output
	} else {
		input, err := NewInput(cfg, devices)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建音频输入失败: %w", err)
		}
		manager.input = input

		output, err := NewOutput(cfg, devices)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建音频输出失败: %w", err)
		}
		manager.output = output
	}

	return manager, nil
}

// StartInput 启动音频输入流
func (m *Manager) StartInput(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("音频管理器已在运行")
	}

	if err := m.input.Start(ctx); err != nil {
		return fmt.Errorf("启动音频输入失败: %w", err)
	}

	m.isRunning = true
	log.Println("音频输入流已启动")
	return nil
}

// StartOutput 启动音频输出流
func (m *Manager) StartOutput(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("音频输入流未启动")
	}

	if err := m.output.Start(ctx); err != nil {
		return fmt.Errorf("启动音频输出失败: %w", err)
	}

	log.Println("音频输出流已启动")
	return nil
}

// Stop 停止音频流
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	if err := m.input.Stop(); err != nil {
		log.Printf("停止音频输入失败: %v", err)
	}

	if err := m.output.Stop(); err != nil {
		log.Printf("停止音频输出失败: %v", err)
	}

	m.isRunning = false
	log.Println("音频流已停止")
	return nil
}

// Close 关闭音频管理器
func (m *Manager) Close() error {
	m.Stop()
	m.cancel()

	if m.input != nil {
		if err := m.input.Close(); err != nil {
			log.Printf("关闭音频输入失败: %v", err)
		}
	}

	if m.output != nil {
		if err := m.output.Close(); err != nil {
			log.Printf("关闭音频输出失败: %v", err)
		}
	}

	if m.devices != nil {
		if err := m.devices.Close(); err != nil {
			log.Printf("关闭设备管理器失败: %v", err)
		}
	}

	return nil
}

// ListDevices 列出可用的音频设备
func (m *Manager) ListDevices() {
	if m.devices != nil {
		m.devices.ListDevices()
	}
}

// GetInputLevel 获取输入音量级别
func (m *Manager) GetInputLevel() float64 {
	if m.input != nil {
		return m.input.GetLevel()
	}
	return 0.0
}

// GetOutputLevel 获取输出音量级别
func (m *Manager) GetOutputLevel() float64 {
	if m.output != nil {
		return m.output.GetLevel()
	}
	return 0.0
}

// SetInputGain 设置输入增益
func (m *Manager) SetInputGain(gain float64) error {
	if m.input != nil {
		return m.input.SetGain(gain)
	}
	return fmt.Errorf("音频输入未初始化")
}

// SetOutputVolume 设置输出音量
func (m *Manager) SetOutputVolume(volume float64) error {
	if m.output != nil {
		return m.output.SetVolume(volume)
	}
	return fmt.Errorf("音频输出未初始化")
}

// IsRunning 检查音频管理器是否正在运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

// UpdateConfig 更新配置
func (m *Manager) UpdateConfig(newConfig *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("无法在运行时更新配置")
	}

	m.config = newConfig

	// 重新初始化音频组件
	if err := m.input.UpdateConfig(newConfig); err != nil {
		return fmt.Errorf("更新输入配置失败: %w", err)
	}

	if err := m.output.UpdateConfig(newConfig); err != nil {
		return fmt.Errorf("更新输出配置失败: %w", err)
	}

	return nil
}

// GetAPRSProcessor 获取APRS音频处理器
func (m *Manager) GetAPRSProcessor() *APRSProcessor {
	return m.aprsProcessor
}

// GetAPRSStatus 获取APRS处理器状态
func (m *Manager) GetAPRSStatus() map[string]interface{} {
	if m.aprsProcessor != nil {
		return m.aprsProcessor.GetStatus()
	}
	return nil
}

// SetAPRSNoiseGate 设置APRS噪声门限
func (m *Manager) SetAPRSNoiseGate(threshold float64) {
	if m.aprsProcessor != nil {
		m.aprsProcessor.SetNoiseGateThreshold(threshold)
	}
}

// SetAPRSCompression 设置APRS压缩比
func (m *Manager) SetAPRSCompression(ratio float64) {
	if m.aprsProcessor != nil {
		m.aprsProcessor.SetCompressionRatio(ratio)
	}
}

// SetAPRSPeakThreshold 设置APRS峰值门限
func (m *Manager) SetAPRSPeakThreshold(threshold float64) {
	if m.aprsProcessor != nil {
		m.aprsProcessor.SetPeakThreshold(threshold)
	}
}
