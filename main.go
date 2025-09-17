package main

import (
	hyperliquid "hyper-notify-bot/hyperLiquid"
	"log"
	"os"
	"os/signal"
	"syscall"

	"hyper-notify-bot/config"
	//"hyper-notify-bot/logger"
	"hyper-notify-bot/scheduler"
	"hyper-notify-bot/service"
	"hyper-notify-bot/telegram"
)

func main() {
	// 初始化日志
	//logger.SetupLogger()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 创建 Hyperliquid WebSocket 客户端
	wsClient := hyperliquid.NewWebSocketClient()
	if err := wsClient.Connect(); err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}
	defer wsClient.Close()

	// 订阅 HYPE 币种
	if err := wsClient.Subscribe("HYPE"); err != nil {
		log.Fatalf("订阅 HYPE 失败: %v", err)
	}
	if err := wsClient.Subscribe("BTC"); err != nil {
		log.Fatalf("订阅 HYPE 失败: %v", err)
	}
	if err := wsClient.Subscribe("SOL"); err != nil {
		log.Fatalf("订阅 HYPE 失败: %v", err)
	}
	if err := wsClient.Subscribe("ETH"); err != nil {
		log.Fatalf("订阅 HYPE 失败: %v", err)
	}

	// 开始监听 WebSocket
	wsClient.StartListening()

	log.Println("已连接 Hyperliquid WebSocket 并订阅 HYPE")

	// 创建数据服务
	dataService, err := service.NewDataService(cfg)
	if err != nil {
		log.Fatalf("创建数据服务失败: %v", err)
	}
	defer dataService.Close()

	// 创建Telegram机器人
	bot := telegram.NewTelegramBot(cfg.TelegramToken, cfg.TelegramChatID, cfg.TelegramProxy)

	// 创建定时任务调度器
	cronScheduler := scheduler.NewCronScheduler(bot, cfg, dataService, wsClient)
	cronScheduler.Start()
	defer cronScheduler.Stop()

	log.Println("Telegram表格数据定时推送系统已启动")
	log.Printf("每 %v 发送一次数据", cfg.Interval)
	log.Printf("数据源: MongoDB (%s/%s)", cfg.MongoDB, cfg.MongoCollection)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("接收到中断信号，程序正在退出...")
}
