package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	configContent := `[audio.input]
device_name = "test_mic"
sample_rate = 48000
channels = 2
buffer_size = 2048
gain = 1.5
format = "int16"

[audio.output]
device_name = "test_speaker"
sample_rate = 48000
channels = 2
buffer_size = 2048
volume = 0.9
format = "int16"

[audio.processing]
echo_cancellation = true
noise_suppression = false
auto_gain_control = true
format = "int16"

[system]
log_level = "debug"
list_devices_on_startup = false
stream_timeout = 3000`

	tmpFile, err := os.CreateTemp("", "test_config_*.conf")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	tmpFile.Close()

	// 测试加载配置
	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if cfg.Audio.Input.DeviceName != "test_mic" {
		t.Errorf("期望输入设备名称为 'test_mic'，实际为 '%s'", cfg.Audio.Input.DeviceName)
	}

	if cfg.Audio.Input.SampleRate != 48000 {
		t.Errorf("期望输入采样率为 48000，实际为 %d", cfg.Audio.Input.SampleRate)
	}

	if cfg.Audio.Input.Channels != 2 {
		t.Errorf("期望输入声道数为 2，实际为 %d", cfg.Audio.Input.Channels)
	}

	if cfg.Audio.Input.BufferSize != 2048 {
		t.Errorf("期望输入缓冲区大小为 2048，实际为 %d", cfg.Audio.Input.BufferSize)
	}

	if cfg.Audio.Input.Gain != 1.5 {
		t.Errorf("期望输入增益为 1.5，实际为 %f", cfg.Audio.Input.Gain)
	}

	if cfg.Audio.Input.Format != "int16" {
		t.Errorf("期望输入格式为 'int16'，实际为 '%s'", cfg.Audio.Input.Format)
	}

	if cfg.Audio.Output.DeviceName != "test_speaker" {
		t.Errorf("期望输出设备名称为 'test_speaker'，实际为 '%s'", cfg.Audio.Output.DeviceName)
	}

	if cfg.Audio.Output.Volume != 0.9 {
		t.Errorf("期望输出音量为 0.9，实际为 %f", cfg.Audio.Output.Volume)
	}

	if cfg.Audio.Processing.EchoCancellation != true {
		t.Errorf("期望回声消除为 true，实际为 %t", cfg.Audio.Processing.EchoCancellation)
	}

	if cfg.Audio.Processing.NoiseSuppression != false {
		t.Errorf("期望噪声抑制为 false，实际为 %t", cfg.Audio.Processing.NoiseSuppression)
	}

	if cfg.System.LogLevel != "debug" {
		t.Errorf("期望日志级别为 'debug'，实际为 '%s'", cfg.System.LogLevel)
	}

	if cfg.System.ListDevicesOnStartup != false {
		t.Errorf("期望启动时列出设备为 false，实际为 %t", cfg.System.ListDevicesOnStartup)
	}

	if cfg.System.StreamTimeout != 3000 {
		t.Errorf("期望流超时时间为 3000，实际为 %d", cfg.System.StreamTimeout)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Gain:       1.0,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     0.8,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "int16",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "无效采样率",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 0,
						Channels:   2,
						BufferSize: 1024,
						Gain:       1.0,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     0.8,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "int16",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "无效声道数",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 44100,
						Channels:   0,
						BufferSize: 1024,
						Gain:       1.0,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     0.8,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "int16",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "无效增益",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Gain:       2.5,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     0.8,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "int16",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "无效音量",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Gain:       1.0,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     1.5,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "int16",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "无效音频格式",
			config: &Config{
				Audio: AudioConfig{
					Input: InputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Gain:       1.0,
						Format:     "int16",
					},
					Output: OutputConfig{
						SampleRate: 44100,
						Channels:   2,
						BufferSize: 1024,
						Volume:     0.8,
						Format:     "int16",
					},
					Processing: ProcessingConfig{
						Format: "invalid",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigGetters(t *testing.T) {
	cfg := &Config{
		Audio: AudioConfig{
			Input: InputConfig{
				SampleRate: 48000,
				Channels:   1,
				BufferSize: 2048,
				Gain:       1.2,
				Format:     "int16",
			},
			Output: OutputConfig{
				SampleRate: 48000,
				Channels:   2,
				BufferSize: 2048,
				Volume:     0.7,
				Format:     "int16",
			},
			Processing: ProcessingConfig{
				EchoCancellation:   true,
				NoiseSuppression:   false,
				AutoGainControl:    true,
				Format:             "int16",
			},
		},
		System: SystemConfig{
			LogLevel:              "warn",
			ListDevicesOnStartup:  false,
			StreamTimeout:         4000,
		},
	}

	// 测试各种getter方法
	if got := cfg.GetSampleRate(); got != 48000 {
		t.Errorf("GetSampleRate() = %v, want %v", got, 48000)
	}

	if got := cfg.GetChannels(); got != 1 {
		t.Errorf("GetChannels() = %v, want %v", got, 1)
	}

	if got := cfg.GetBufferSize(); got != 2048 {
		t.Errorf("GetBufferSize() = %v, want %v", got, 2048)
	}

	if got := cfg.GetGain(); got != 1.2 {
		t.Errorf("GetGain() = %v, want %v", got, 1.2)
	}

	if got := cfg.GetVolume(); got != 0.7 {
		t.Errorf("GetVolume() = %v, want %v", got, 0.7)
	}

	if got := cfg.GetFormat(); got != "int16" {
		t.Errorf("GetFormat() = %v, want %v", got, "int16")
	}

	if got := cfg.IsEchoCancellationEnabled(); got != true {
		t.Errorf("IsEchoCancellationEnabled() = %v, want %v", got, true)
	}

	if got := cfg.IsNoiseSuppressionEnabled(); got != false {
		t.Errorf("IsNoiseSuppressionEnabled() = %v, want %v", got, false)
	}

	if got := cfg.IsAutoGainControlEnabled(); got != true {
		t.Errorf("IsAutoGainControlEnabled() = %v, want %v", got, true)
	}

	if got := cfg.GetLogLevel(); got != "warn" {
		t.Errorf("GetLogLevel() = %v, want %v", got, "warn")
	}

	if got := cfg.ShouldListDevicesOnStartup(); got != false {
		t.Errorf("ShouldListDevicesOnStartup() = %v, want %v", got, false)
	}

	if got := cfg.GetStreamTimeout(); got != 4000 {
		t.Errorf("GetStreamTimeout() = %v, want %v", got, 4000)
	}
}
