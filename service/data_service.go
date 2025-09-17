package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"hyper-notify-bot/config"
	"hyper-notify-bot/db"
)

// DataService 管理数据获取
type DataService struct {
	DBClient *mongodb.MongoDBClient
	Config   *config.Config
}

// NewDataService 创建新的数据服务
func NewDataService(cfg *config.Config) (*DataService, error) {
	dbClient, err := mongodb.NewMongoDBClient(cfg)
	if err != nil {
		return nil, err
	}

	return &DataService{
		DBClient: dbClient,
		Config:   cfg,
	}, nil
}

// Close 关闭数据服务
func (ds *DataService) Close() {
	ds.DBClient.Close()
}

// GetTableData 获取表格数据（带重试机制）
func (ds *DataService) GetTableData(coin, oraclePriceStr string) ([]mongodb.PositionResult, float64, float64, error) {
	var lastErr error

	minPrice := float64(47)
	maxPrice := float64(53)

	if oraclePriceStr != "N/A" {
		oraclePrice, _ := strconv.ParseFloat(oraclePriceStr, 64)
		ratio := ds.Config.PriceRangeRatio
		minPrice = oraclePrice * (1 - ratio)
		maxPrice = oraclePrice * (1 + ratio)
	}

	for i := 0; i < ds.Config.RetryCount; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		longSz, shortSz, err := ds.DBClient.GetPositionSummary(ctx, coin)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("获取数据失败，已达最大重试次数: %v", err)
		}

		fmt.Println(longSz, shortSz)

		data, err := ds.DBClient.GetPricePositionSummary(ctx, coin, minPrice, maxPrice)
		if err == nil {
			return data, longSz, shortSz, nil
		}

		lastErr = err
		log.Printf("获取数据失败 (尝试 %d/%d): %v", i+1, ds.Config.RetryCount, err)
		time.Sleep(ds.Config.RetryDelay)
	}

	return nil, 0, 0, fmt.Errorf("获取数据失败，已达最大重试次数: %v", lastErr)
}
