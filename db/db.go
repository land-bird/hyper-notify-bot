package mongodb

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hyper-notify-bot/config"
)

// MongoDBClient 封装MongoDB客户端
type MongoDBClient struct {
	Client     *mongo.Client
	Database   *mongo.Database
	Collection *mongo.Collection
}

// NewMongoDBClient 创建并初始化MongoDB客户端
func NewMongoDBClient(cfg *config.Config) (*MongoDBClient, error) {
	// 设置客户端选项
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)

	// 设置连接超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接到MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("连接MongoDB失败: %v", err)
	}

	// 检查连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("MongoDB连接验证失败: %v", err)
	}

	log.Println("成功连接到MongoDB")

	// 获取数据库和集合
	database := client.Database(cfg.MongoDB)
	collection := database.Collection(cfg.MongoCollection)

	return &MongoDBClient{
		Client:     client,
		Database:   database,
		Collection: collection,
	}, nil
}

// Close 关闭MongoDB连接
func (m *MongoDBClient) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.Client.Disconnect(ctx); err != nil {
		log.Printf("关闭MongoDB连接失败: %v", err)
	} else {
		log.Println("MongoDB连接已关闭")
	}
}

// GetTableData 从MongoDB获取表格数据
func (m *MongoDBClient) GetTableData(ctx context.Context) ([]TableRow, error) {
	// 设置查询超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 查询所有数据（按id排序）
	findOptions := options.Find().SetSort(bson.D{{Key: "id", Value: 1}})
	cursor, err := m.Collection.Find(ctx, bson.D{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("查询MongoDB失败: %v", err)
	}
	defer cursor.Close(ctx)

	// 解码结果
	var results []TableRow
	for cursor.Next(ctx) {
		var row TableRow
		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("解码文档失败: %v", err)
		}
		results = append(results, row)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("游标错误: %v", err)
	}

	return results, nil
}

// GetPricePositionSummary 获取仓位汇总数据
func (m *MongoDBClient) GetPricePositionSummary(ctx context.Context, coin string, min, max float64) ([]PositionResult, error) {
	// 初始化一个带有几个元素的map
	coinDiv := map[string]float32{
		"BTC":  100,
		"ETH":  10,
		"SOL":  1,
		"HYPE": 0.5,
	}
	// 创建聚合管道
	pipeline := mongo.Pipeline{
		// 1. 筛选价格区间
		{{"$match", bson.D{
			{"px", bson.D{
				{"$gte", min},
				{"$lt", max},
			}},
		}}},

		// 2. 计算所属区间（bin）
		{{"$addFields", bson.D{
			{"bin", bson.D{
				{"$trunc", bson.D{
					{"$divide", bson.A{"$px", coinDiv[coin]}},
				}},
			}},
		}}},

		// 3. 按区间 + Long/Short 汇总
		{{"$group", bson.D{
			{"_id", bson.D{
				{"bin", "$bin"},
				{"dir", "$dir"},
			}},
			{"totalSz", bson.D{{"$sum", "$sz"}}},
		}}},

		// 4. 按 bin 分组
		{{"$group", bson.D{
			{"_id", "$_id.bin"},
			{"positions", bson.D{
				{"$push", bson.D{
					{"dir", "$_id.dir"},
					{"totalSz", "$totalSz"},
				}},
			}},
		}}},

		// 5. 转换为对象格式
		{{"$project", bson.D{
			{"bin", bson.D{
				{"$multiply", bson.A{"$_id", coinDiv[coin]}},
			}},
			{"positions", bson.D{
				{"$arrayToObject", bson.D{
					{"$map", bson.D{
						{"input", "$positions"},
						{"as", "pos"},
						{"in", bson.D{
							{"k", "$$pos.dir"},
							{"v", "$$pos.totalSz"},
						}},
					}},
				}},
			}},
		}}},

		// 6. 最终格式
		{{"$project", bson.D{
			{"bin", 1},
			{"Long", bson.D{
				{"$convert", bson.D{
					{"input", "$positions.Long"},
					{"to", "decimal"},
					{"onError", 0},
					// {"onNull", 0}, // 如果需要处理 null，取消注释
				}},
			}},
			{"Short", bson.D{{"$convert", bson.D{ // 使用 $convert 操作符
				{"input", "$positions.Short"}, // 输入字段
				{"to", "decimal"},             // 转换为 decimal 类型
				{"onError", 0},                // 转换失败时的默认值
			}}}},
		}}},

		// 7. 排序
		{{"$sort", bson.D{{"bin", 1}}}},
	}

	// 添加超时控制
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 执行聚合查询
	//m.Database.Collection("hype_positions")
	cursor, err := m.Database.Collection(strings.ToLower(coin)+"_positions").Aggregate(ctx, pipeline, options.Aggregate().SetMaxTime(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("聚合查询失败: %v", err)
	}
	defer cursor.Close(ctx)

	// 解析结果
	var results []PositionResult
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("结果解析失败: %v", err)
	}

	return results, nil
}

func (m *MongoDBClient) GetPositionSummary(ctx context.Context, coin string) (float64, float64, error) {
	// 构建聚合管道 - 返回单行，Long 和 Short 作为列
	pipeline := mongo.Pipeline{
		// 匹配阶段: 筛选 dir 为 "Short" 或 "Long" 的文档
		{{Key: "$match", Value: bson.D{
			{Key: "dir", Value: bson.D{{
				Key: "$in", Value: bson.A{"Short", "Long"},
			}}},
		}}},
		// 分组阶段: 将所有文档归入同一组，并分别计算 Long 和 Short 的总和
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "Long", Value: bson.D{
				{Key: "$sum", Value: bson.D{
					{Key: "$cond", Value: bson.A{
						bson.D{{Key: "$eq", Value: bson.A{"$dir", "Long"}}},
						"$sz",
						0,
					}},
				}},
			}},
			{Key: "Short", Value: bson.D{
				{Key: "$sum", Value: bson.D{
					{Key: "$cond", Value: bson.A{
						bson.D{{Key: "$eq", Value: bson.A{"$dir", "Short"}}},
						"$sz",
						0,
					}},
				}},
			}}},
		}},
		// 投影阶段: 移除 _id 字段
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "Long", Value: 1},
			{Key: "Short", Value: 1},
		}}},
	}

	// 执行聚合查询
	cursor, err := m.Database.Collection(strings.ToLower(coin)+"_positions").Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, fmt.Errorf("聚合查询失败: %v", err)
	}
	defer cursor.Close(ctx)

	// 解码结果到结构体
	var results []PositionTotals
	if err = cursor.All(ctx, &results); err != nil {
		return 0, 0, fmt.Errorf("结果解码失败: %v", err)
	}

	// 检查是否有结果
	if len(results) == 0 {
		return 0, 0, nil // 没有匹配的数据，返回零值
	}

	longSz, _ := strconv.ParseFloat(results[0].Long.String(), 64)
	shortSz, _ := strconv.ParseFloat(results[0].Short.String(), 64)
	// 返回 Long 和 Short 的值
	return longSz, shortSz, nil
}
