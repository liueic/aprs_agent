//go:build !darwin

package audio

import (
	"aprs_agent/config"
	"context"
	"fmt"
)

// genericOutput 通用音频输出存根
type genericOutput struct {
	config  *config.Config
	devices DeviceManagerInterface
}

// newGenericOutput 创建通用音频输出
func newGenericOutput(cfg *config.Config, devices DeviceManagerInterface) (AudioOutput, error) {
	return &genericOutput{
		config:  cfg,
		devices: devices,
	}, nil
}

// 实现AudioOutput接口
func (g *genericOutput) Start(ctx context.Context) error {
	return fmt.Errorf("通用音频输出暂未实现，请使用平台专用版本")
}

func (g *genericOutput) Stop() error                                 { return nil }
func (g *genericOutput) Close() error                                { return nil }
func (g *genericOutput) PlayAudio(data []byte) error                 { return nil }
func (g *genericOutput) GetLevel() float64                           { return 0.0 }
func (g *genericOutput) SetVolume(volume float64) error              { return nil }
func (g *genericOutput) GetVolume() float64                          { return 1.0 }
func (g *genericOutput) IsRunning() bool                             { return false }
func (g *genericOutput) UpdateConfig(newConfig *config.Config) error { return nil }
func (g *genericOutput) GetBuffer() []byte                           { return nil }
func (g *genericOutput) GetConfig() *config.Config                   { return g.config }
func (g *genericOutput) GetQueueSize() int                           { return 0 }
func (g *genericOutput) ClearQueue()                                 {}
