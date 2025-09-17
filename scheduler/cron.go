package scheduler

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"hyper-notify-bot/config"
	"hyper-notify-bot/formatter"
	hyperliquid "hyper-notify-bot/hyperLiquid"
	"hyper-notify-bot/service"
	"hyper-notify-bot/telegram"
	"log"
)

type CronScheduler struct {
	Cron        *cron.Cron
	Bot         *telegram.TelegramBot
	Config      *config.Config
	DataService *service.DataService
	WsClient    *hyperliquid.WebSocketClient
}

func NewCronScheduler(bot *telegram.TelegramBot,
	cfg *config.Config,
	dataService *service.DataService,
	wsClient *hyperliquid.WebSocketClient) *CronScheduler {
	return &CronScheduler{
		Cron:        cron.New(cron.WithSeconds()),
		Bot:         bot,
		Config:      cfg,
		DataService: dataService,
		WsClient:    wsClient,
	}
}

func (s *CronScheduler) Start() {
	// 添加定时任务
	_, err := s.Cron.AddFunc(fmt.Sprintf("0 */%d * * * *", int(s.Config.Interval.Minutes())), s.sendTableJob)
	if err != nil {
		log.Fatalf("添加定时任务失败: %v", err)
	}

	s.Cron.Start()
	log.Println("定时任务调度器已启动")
}

func (s *CronScheduler) Stop() {
	s.Cron.Stop()
	s.DataService.Close()
	log.Println("定时任务调度器已停止")
}

func (s *CronScheduler) sendTableJob() {
	s.sendCoinTableJob("HYPE")
	s.sendCoinTableJob("BTC")
	s.sendCoinTableJob("ETH")
	s.sendCoinTableJob("SOL")
}

func (s *CronScheduler) sendCoinTableJob(coin string) {
	log.Println("开始执行定时任务...从MongoDB读取数据")

	// 获取最新的 Oracle 价格
	var oraclePrice string
	if price, exists := s.WsClient.GetOraclePrice(coin); exists {
		oraclePrice = price.OraclePx
		log.Printf("当前 %s Oracle 价格: %s", coin, oraclePrice)
	} else {
		oraclePrice = "N/A"
		log.Printf("未找到 %s 的 Oracle 价格", coin)
	}

	// 从服务层获取数据
	data, longSz, shortSz, err := s.DataService.GetTableData(coin, oraclePrice)
	if err != nil {
		log.Printf("获取数据失败: %v", err)
		return
	}

	// 格式化消息
	message := formatter.FormatTableAsHTML(data, coin, oraclePrice, longSz, shortSz)

	// 发送消息
	if err := s.Bot.SendWithRetry(context.Background(), message, "HTML", s.Config); err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Printf("成功发送 %d 行数据到Telegram", len(data))
	}
}
