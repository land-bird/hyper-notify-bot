package hyperliquid

import "time"

// WebSocketRequest 订阅请求
type SubscribeRequest struct {
	Method       string       `json:"method"`
	Subscription Subscription `json:"subscription"`
}

type Subscription struct {
	Type string `json:"type"`
	Coin string `json:"coin"`
}

// WebSocketResponse WebSocket 响应
type WebSocketResponse struct {
	Channel string `json:"channel"`
	Data    struct {
		Coin string `json:"coin"`
		Ctx  struct {
			Funding      string   `json:"funding"`
			OpenInterest string   `json:"openInterest"`
			PrevDayPx    string   `json:"prevDayPx"`
			DayNtlVlm    string   `json:"dayNtlVlm"`
			Premium      string   `json:"premium"`
			OraclePx     string   `json:"oraclePx"` // 这是我们需要的关键字段
			MarkPx       string   `json:"markPx"`
			MidPx        string   `json:"midPx"`
			ImpactPxs    []string `json:"impactPxs"`
			DayBaseVlm   string   `json:"dayBaseVlm"`
		} `json:"ctx"`
	} `json:"data"`
}

// OraclePrice 存储 Oracle 价格
type OraclePrice struct {
	Coin      string    `json:"coin"`
	OraclePx  string    `json:"oraclePx"`
	Timestamp time.Time `json:"timestamp"`
}
