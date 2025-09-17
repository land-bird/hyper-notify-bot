package hyperliquid

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	WebSocketURL   = "wss://api.hyperliquid.xyz/ws"
	ReconnectDelay = 5 * time.Second
)

type WebSocketClient struct {
	conn         *websocket.Conn
	mu           sync.RWMutex
	oraclePrices map[string]OraclePrice // coin -> OraclePrice
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewWebSocketClient() *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketClient{
		oraclePrices: make(map[string]OraclePrice),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (c *WebSocketClient) Connect() error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(WebSocketURL, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *WebSocketClient) Subscribe(coin string) error {
	request := SubscribeRequest{
		Method: "subscribe",
		Subscription: Subscription{
			Type: "activeAssetCtx",
			Coin: coin,
		},
	}

	message, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, message)
}

func (c *WebSocketClient) StartListening() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				if c.conn == nil {
					if err := c.Connect(); err != nil {
						log.Printf("连接失败，将在 %v 后重试: %v", ReconnectDelay, err)
						time.Sleep(ReconnectDelay)
						continue
					}

					// 重新订阅所有币种
					c.mu.RLock()
					coins := make([]string, 0, len(c.oraclePrices))
					for coin := range c.oraclePrices {
						coins = append(coins, coin)
					}
					c.mu.RUnlock()

					for _, coin := range coins {
						if err := c.Subscribe(coin); err != nil {
							log.Printf("重新订阅 %s 失败: %v", coin, err)
						}
					}
				}

				_, message, err := c.conn.ReadMessage()
				if err != nil {
					log.Printf("读取消息错误: %v", err)
					c.conn.Close()
					c.conn = nil
					continue
				}

				var response WebSocketResponse
				if err := json.Unmarshal(message, &response); err != nil {
					log.Printf("解析消息错误: %v", err)
					continue
				}

				if response.Channel == "activeAssetCtx" {
					oraclePrice := OraclePrice{
						Coin:      response.Data.Coin,
						OraclePx:  response.Data.Ctx.OraclePx,
						Timestamp: time.Now(),
					}

					c.mu.Lock()
					c.oraclePrices[response.Data.Coin] = oraclePrice
					c.mu.Unlock()

					log.Printf("更新 %s 的 Oracle 价格: %s", response.Data.Coin, oraclePrice.OraclePx)
				}
			}
		}
	}()
}

func (c *WebSocketClient) GetOraclePrice(coin string) (OraclePrice, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	price, exists := c.oraclePrices[coin]
	return price, exists
}

func (c *WebSocketClient) Close() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
}
