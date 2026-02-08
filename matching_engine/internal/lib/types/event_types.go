package types

import (
	"encoding/json"
	"time"
)

type EventType int64

const (
	OrderPlaced EventType = iota
	OrderCancelled
	OrderFilled
	OrderPartiallyFilled
	OrderRejected
	TradeExecuted
	PriceChanged
)

type Event struct {
	EventID   string          `json:"event_id"`
	Timestamp time.Time       `json:"timestamp"`
	Type      EventType       `json:"type"`
	Data      json.RawMessage `json:"data"`
}

type OrderPlacedEvent struct {
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	BotID           string    `json:"bot_id"`
	StockTicker     string    `json:"stock_ticker"`
	OrderType       OrderType `json:"order_type"`
	OrderSide       OrderSide `json:"order_side"`
	Quantity        int64     `json:"quantity"`
	LimitPriceCents int64     `json:"limit_price_cents"`
}

type OrderCancelledEvent struct {
	OrderID           string `json:"order_id"`
	UserID            string `json:"user_id"`
	BotID             string `json:"bot_id"`
	RemainingQuantity int64  `json:"remaining_quantity"`
}

type OrderFilledEvent struct {
	OrderID               string `json:"order_id"`
	UserID                string `json:"user_id"`
	BotID                 string `json:"bot_id"`
	TotalQuantity         int64  `json:"total_quantity"`
	AverageFillPriceCents int64  `json:"average_fill_price_cents"`
}

type OrderPartiallyFilledEvent struct {
	OrderID           string `json:"order_id"`
	UserID            string `json:"user_id"`
	BotID             string `json:"bot_id"`
	FilledQuantity    int64  `json:"filled_quantity"`
	RemainingQuantity int64  `json:"remaining_quantity"`
	FillPriceCents    int64  `json:"fill_price_cents"`
}

type OrderRejectedEvent struct {
	OrderID      string `json:"order_id"`
	UserID       string `json:"user_id"`
	BotID        string `json:"bot_id"`
	Reason       string `json:"reason"`
	ErrorMessage string `json:"error_message"`
}

type TradeExecutedEvent struct {
	TradeID         string `json:"trade_id"`
	StockID         string `json:"stock_id"`
	BuyerOrderID    string `json:"buyer_order_id"`
	SellerOrderID   string `json:"seller_order_id"`
	BuyerUserID     string `json:"buyer_user_id"`
	BuyerBotID      string `json:"buyer_bot_id"`
	SellerUserID    string `json:"seller_user_id"`
	SellerBotID     string `json:"seller_bot_id"`
	Quantity        int64  `json:"quantity"`
	PriceCents      int64  `json:"price_cents"`
	TotalValueCents int64  `json:"total_value_cents"`
}

type PriceChangedEvent struct {
	StockID         string `json:"stock_id"`
	OldPriceCents   int64  `json:"old_price_cents"`
	NewPriceCents   int64  `json:"new_price_cents"`
	CausedByTradeID string `json:"caused_by_trade_id"`
}
