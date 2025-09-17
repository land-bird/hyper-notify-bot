package mongodb

import "go.mongodb.org/mongo-driver/bson/primitive"

// TableRow 表示从MongoDB读取的数据结构
type TableRow struct {
	ID  float64 `bson:"id"`  // 编号
	Pos float64 `bson:"pos"` // 正值
	Neg float64 `bson:"neg"` // 负值
}

// PositionResult 定义查询结果结构
type PositionResult struct {
	Bin   primitive.Decimal128 `bson:"bin"`
	Long  primitive.Decimal128 `bson:"Long"`
	Short primitive.Decimal128 `bson:"Short"`
}

type PositionTotals struct {
	Long  primitive.Decimal128 `bson:"Long"`
	Short primitive.Decimal128 `bson:"Short"`
}
