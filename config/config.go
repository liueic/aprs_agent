package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 表示应用程序的配置结构
type Config struct {
	Audio  AudioConfig  `mapstructure:"audio"`
	System SystemConfig `mapstructure:"system"`
}

// AudioConfig 音频相关配置
type AudioConfig struct {
	Input      InputConfig      `mapstructure:"input"`
	Output     OutputConfig     `mapstructure:"output"`
	Processing ProcessingConfig `mapstructure:"processing"`
}

// InputConfig 输入音频配置
type InputConfig struct {
	DeviceName string  `mapstructure:"device_name"`
	SampleRate int     `mapstructure:"sample_rate"`
	Channels   int     `mapstructure:"channels"`
	BufferSize int     `mapstructure:"buffer_size"`
	Gain       float64 `mapstructure:"gain"`
	Format     string  `mapstructure:"format"`
}

// OutputConfig 输出音频配置
type OutputConfig struct {
	DeviceName string  `mapstructure:"device_name"`
	SampleRate int     `mapstructure:"sample_rate"`
	Channels   int     `mapstructure:"channels"`
	BufferSize int     `mapstructure:"buffer_size"`
	Volume     float64 `mapstructure:"volume"`
	Format     string  `mapstructure:"format"`
}

// ProcessingConfig 音频处理配置
type ProcessingConfig struct {
	EchoCancellation bool   `mapstructure:"echo_cancellation"`
	NoiseSuppression bool   `mapstructure:"noise_suppression"`
	AutoGainControl  bool   `mapstructure:"auto_gain_control"`
	Format           string `mapstructure:"format"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	LogLevel             string `mapstructure:"log_level"`
	ListDevicesOnStartup bool   `mapstructure:"list_devices_on_startup"`
	StreamTimeout        int    `mapstructure:"stream_timeout"`
	APRSMode             bool   `mapstructure:"aprs_mode"`
	LevelMonitorInterval int    `mapstructure:"level_monitor_interval"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	viper.SetConfigFile(filename)
	viper.SetConfigType("ini")

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 音频输入默认值 (APRS优化)
	viper.SetDefault("audio.input.sample_rate", 8000)
	viper.SetDefault("audio.input.channels", 1)
	viper.SetDefault("audio.input.buffer_size", 256)
	viper.SetDefault("audio.input.gain", 1.2)

	// 音频输出默认值 (APRS优化)
	viper.SetDefault("audio.output.sample_rate", 8000)
	viper.SetDefault("audio.output.channels", 1)
	viper.SetDefault("audio.output.buffer_size", 256)
	viper.SetDefault("audio.output.volume", 0.8)

	// 音频处理默认值 (APRS优化)
	viper.SetDefault("audio.processing.echo_cancellation", false)
	viper.SetDefault("audio.processing.noise_suppression", true)
	viper.SetDefault("audio.processing.auto_gain_control", true)
	viper.SetDefault("audio.processing.format", "int16")

	// 系统默认值
	viper.SetDefault("system.log_level", "info")
	viper.SetDefault("system.list_devices_on_startup", true)
	viper.SetDefault("system.stream_timeout", 2000)
	viper.SetDefault("system.aprs_mode", true)
	viper.SetDefault("system.level_monitor_interval", 100)
}

// validateConfig 验证配置的有效性
func validateConfig(config *Config) error {
	// 验证采样率
	if config.Audio.Input.SampleRate <= 0 {
		return fmt.Errorf("输入采样率必须大于0")
	}
	if config.Audio.Output.SampleRate <= 0 {
		return fmt.Errorf("输出采样率必须大于0")
	}

	// 验证声道数
	if config.Audio.Input.Channels <= 0 || config.Audio.Input.Channels > 8 {
		return fmt.Errorf("输入声道数必须在1-8之间")
	}
	if config.Audio.Output.Channels <= 0 || config.Audio.Output.Channels > 8 {
		return fmt.Errorf("输出声道数必须在1-8之间")
	}

	// 验证缓冲区大小
	if config.Audio.Input.BufferSize <= 0 {
		return fmt.Errorf("输入缓冲区大小必须大于0")
	}
	if config.Audio.Output.BufferSize <= 0 {
		return fmt.Errorf("输出缓冲区大小必须大于0")
	}

	// 验证增益和音量
	if config.Audio.Input.Gain < 0 || config.Audio.Input.Gain > 2 {
		return fmt.Errorf("输入增益必须在0.0-2.0之间")
	}
	if config.Audio.Output.Volume < 0 || config.Audio.Output.Volume > 1 {
		return fmt.Errorf("输出音量必须在0.0-1.0之间")
	}

	// 验证音频格式
	if config.Audio.Processing.Format != "int16" && config.Audio.Processing.Format != "float32" {
		return fmt.Errorf("音频格式必须是 'int16' 或 'float32'")
	}

	return nil
}

// GetString 获取字符串配置值
func (c *Config) GetString(key string) string {
	return viper.GetString(key)
}

// GetInt 获取整数配置值
func (c *Config) GetInt(key string) int {
	return viper.GetInt(key)
}

// GetFloat64 获取浮点数配置值
func (c *Config) GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

// GetBool 获取布尔配置值
func (c *Config) GetBool(key string) bool {
	return viper.GetBool(key)
}

// Set 设置配置值
func (c *Config) Set(key string, value interface{}) {
	viper.Set(key, value)
}

// WriteConfig 将配置写入文件
func (c *Config) WriteConfig() error {
	return viper.WriteConfig()
}

// GetSampleRate 获取采样率
func (c *Config) GetSampleRate() int {
	return c.Audio.Input.SampleRate
}

// GetChannels 获取声道数
func (c *Config) GetChannels() int {
	return c.Audio.Input.Channels
}

// GetBufferSize 获取缓冲区大小
func (c *Config) GetBufferSize() int {
	return c.Audio.Input.BufferSize
}

// GetGain 获取增益
func (c *Config) GetGain() float64 {
	return c.Audio.Input.Gain
}

// GetVolume 获取音量
func (c *Config) GetVolume() float64 {
	return c.Audio.Output.Volume
}

// GetFormat 获取音频格式
func (c *Config) GetFormat() string {
	return c.Audio.Processing.Format
}

// IsEchoCancellationEnabled 是否启用回声消除
func (c *Config) IsEchoCancellationEnabled() bool {
	return c.Audio.Processing.EchoCancellation
}

// IsNoiseSuppressionEnabled 是否启用噪声抑制
func (c *Config) IsNoiseSuppressionEnabled() bool {
	return c.Audio.Processing.NoiseSuppression
}

// IsAutoGainControlEnabled 是否启用自动增益控制
func (c *Config) IsAutoGainControlEnabled() bool {
	return c.Audio.Processing.AutoGainControl
}

// GetLogLevel 获取日志级别
func (c *Config) GetLogLevel() string {
	return c.System.LogLevel
}

// ShouldListDevicesOnStartup 是否在启动时列出设备
func (c *Config) ShouldListDevicesOnStartup() bool {
	return c.System.ListDevicesOnStartup
}

// GetStreamTimeout 获取流超时时间
func (c *Config) GetStreamTimeout() int {
	return c.System.StreamTimeout
}

// IsAPRSMode 是否为APRS模式
func (c *Config) IsAPRSMode() bool {
	return c.System.APRSMode
}

// GetLevelMonitorInterval 获取音频电平监控间隔
func (c *Config) GetLevelMonitorInterval() int {
	return c.System.LevelMonitorInterval
}
