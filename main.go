package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"aprs_agent/audio"
	"aprs_agent/config"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("app.conf")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建音频管理器
	audioManager, err := audio.NewManager(cfg)
	if err != nil {
		log.Fatalf("创建音频管理器失败: %v", err)
	}
	defer audioManager.Close()

	// 列出可用设备
	if cfg.System.ListDevicesOnStartup {
		audioManager.ListDevices()
	}

	// 启动音频流
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动音频输入流
	if err := audioManager.StartInput(ctx); err != nil {
		log.Fatalf("启动音频输入失败: %v", err)
	}

	// 启动音频输出流
	if err := audioManager.StartOutput(ctx); err != nil {
		log.Fatalf("启动音频输出失败: %v", err)
	}

	fmt.Println("音频系统已启动，按 Ctrl+C 退出...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在关闭音频系统...")
	cancel()
}
