package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken  string
	TelegramChatID string
	TelegramProxy  string
	Interval       time.Duration
	RetryCount     int
	RetryDelay     time.Duration

	// MongoDB配置
	MongoURI        string
	MongoDB         string
	MongoCollection string

	HyperliquidCoin string
	PriceRangeRatio float64 `json:"price_range_ratio"`
}

func LoadConfig() (*Config, error) {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("加载.env文件失败: %v", err)
	}

	intervalStr := os.Getenv("INTERVAL")
	var interval time.Duration
	var err error

	if intervalStr != "" {
		// 尝试解析复杂持续时间格式（如 "1h30m"）
		interval, err = time.ParseDuration(intervalStr)
		if err != nil {
			// 如果解析失败，尝试作为纯数字分钟数解析
			intervalVal, err2 := strconv.Atoi(intervalStr)
			if err2 != nil {
				// 如果两种方式都失败，使用默认值
				interval = 1 * time.Minute
			} else {
				interval = time.Duration(intervalVal) * time.Minute
			}
		}
	} else {
		interval = 1 * time.Minute
	}

	return &Config{
		TelegramToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:  os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramProxy:   os.Getenv("TELEGRAM_PROXY"),
		MongoURI:        os.Getenv("MONGO_URI"),
		MongoDB:         os.Getenv("MONGO_DB"),
		MongoCollection: os.Getenv("MONGO_COLLECTION"),
		HyperliquidCoin: os.Getenv("HYPERLIQUID_COIN"),
		Interval:        interval,        // 每interval分钟执行一次
		RetryCount:      3,               // 最大重试次数
		RetryDelay:      5 * time.Second, // 重试延迟
		PriceRangeRatio: 0.05,
	}, nil

}
