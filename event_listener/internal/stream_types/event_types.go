package streamtypes

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
)

type Event struct {
	EventID   string          `json:"event_id"`
	Timestamp time.Time       `json:"timestamp"`
	Type      EventType       `json:"type"`
	Data      json.RawMessage `json:"data"`
}

type OrderPlacedEvent struct {
	OrderID         string    `json:"order_id"`
	TraderID        int64     `json:"trader_id"`
	StockTicker     string    `json:"stock_ticker"`
	OrderType       OrderType `json:"order_type"`
	OrderSide       OrderSide `json:"order_side"`
	Quantity        int64     `json:"quantity"`
	LimitPriceCents int64     `json:"limit_price_cents"`
}

type OrderCancelledEvent struct {
	OrderID           string    `json:"order_id"`
	TraderID          int64     `json:"trader_id"`
	OrderType         OrderType `json:"order_type"`
	OrderSide         OrderSide `json:"order_side"`
	StockTicker       string    `json:"stock_ticker"`
	RemainingQuantity int64     `json:"remaining_quantity"`
}

type OrderFilledEvent struct {
	OrderID        string `json:"order_id"`
	TraderID       int64  `json:"trader_id"`
	Quantity       int64  `json:"total_quantity"`
	FillPriceCents int64  `json:"fill_price_cents"`
}

type OrderPartiallyFilledEvent struct {
	OrderID           string `json:"order_id"`
	TraderID          int64  `json:"trader_id"`
	FilledQuantity    int64  `json:"filled_quantity"`
	RemainingQuantity int64  `json:"remaining_quantity"`
	FillPriceCents    int64  `json:"fill_price_cents"`
}

type OrderRejectedEvent struct {
	OrderID      string `json:"order_id"`
	TraderID     int64  `json:"trader_id"`
	Reason       string `json:"reason"`
	ErrorMessage string `json:"error_message"`
}

type TradeExecutedEvent struct {
	StockTicker     string    `json:"stock_ticker"`
	BuyerOrderID    string    `json:"buyer_order_id"`
	SellerOrderID   string    `json:"seller_order_id"`
	BuyerOrderType  OrderType `json:"buyer_order_type"`
	BuyerTraderID   int64     `json:"buyer_trader_id"`
	SellerTraderID  int64     `json:"seller_trader_id"`
	Quantity        int64     `json:"quantity"`
	PriceCents      int64     `json:"price_cents"`
	TotalValueCents int64     `json:"total_value_cents"`
}

// EventPayload is a marker interface that all event payload types implement
type EventPayload interface {
	eventPayload() // unexported method ensures only this package can implement it
}

// Implement EventPayload for all event types
func (*OrderPlacedEvent) eventPayload()          {}
func (*OrderCancelledEvent) eventPayload()       {}
func (*OrderFilledEvent) eventPayload()          {}
func (*OrderPartiallyFilledEvent) eventPayload() {}
func (*OrderRejectedEvent) eventPayload()        {}
func (*TradeExecutedEvent) eventPayload()        {}
