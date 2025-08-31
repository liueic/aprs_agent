//go:build !darwin

package audio

import (
	"aprs_agent/config"
	"context"
	"fmt"
)

// genericInput 通用音频输入存根
type genericInput struct {
	config  *config.Config
	devices DeviceManagerInterface
}

// newGenericInput 创建通用音频输入
func newGenericInput(cfg *config.Config, devices DeviceManagerInterface) (AudioInput, error) {
	return &genericInput{
		config:  cfg,
		devices: devices,
	}, nil
}

// 实现AudioInput接口
func (g *genericInput) Start(ctx context.Context) error {
	return fmt.Errorf("通用音频输入暂未实现，请使用平台专用版本")
}

func (g *genericInput) Stop() error                                 { return nil }
func (g *genericInput) Close() error                                { return nil }
func (g *genericInput) GetLevel() float64                           { return 0.0 }
func (g *genericInput) SetGain(gain float64) error                  { return nil }
func (g *genericInput) GetGain() float64                            { return 1.0 }
func (g *genericInput) SetCallback(callback func([]byte, int))      {}
func (g *genericInput) IsRunning() bool                             { return false }
func (g *genericInput) UpdateConfig(newConfig *config.Config) error { return nil }
func (g *genericInput) GetBuffer() []byte                           { return nil }
func (g *genericInput) GetConfig() *config.Config                   { return g.config }
