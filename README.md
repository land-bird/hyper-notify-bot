## How to use

**Prerequisite:** go version>=1.23

1. move .env.example .env, Modify your configuration items
```
# Telegram配置
TELEGRAM_BOT_TOKEN=YOUR_TELEGRAM_BOT_TOKEN
TELEGRAM_CHAT_ID=YOUR_TELEGRAM_CHANNEL_ID
TELEGRAM_PROXY=

# MongoDB配置
MONGO_URI=mongodb://user:pass@localhost:27017
MONGO_DB=your db
MONGO_COLLECTION=hype_positions

# Hyperliquid配置
HYPERLIQUID_COIN=HYPE

#定时任务间隔, 设置数字表示分钟，也可用1h30m,30m
INTERVAL=1h
```
2. build
```
go mod tidy
go build
```
3. run
```
./hyper-notify-bot
```
