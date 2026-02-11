package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/Marwan051/tradding_platform_game/event_listener/internal/db/postgres/out"
	streamtypes "github.com/Marwan051/tradding_platform_game/event_listener/internal/stream_types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresDB struct {
	db db.Queries
}

func New(queries *db.Queries) *PostgresDB {
	return &PostgresDB{db: *queries}
}

func orderTypeToString(t streamtypes.OrderType) string {
	switch t {
	case streamtypes.MarketOrder:
		return "MARKET"
	case streamtypes.LimitOrder:
		return "LIMIT"
	default:
		return ""
	}
}

func orderSideToString(s streamtypes.OrderSide) string {
	switch s {
	case streamtypes.Buy:
		return "BUY"
	case streamtypes.Sell:
		return "SELL"
	default:
		return ""
	}
}

func userIDToPgType(userID string) pgtype.Text {
	if userID == "" {
		return pgtype.Text{Valid: false}
	}

	return pgtype.Text{String: userID, Valid: true}
}

func orderIDToUUID(orderID string) (pgtype.UUID, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("could not parse order UUID: %w", err)
	}
	return pgtype.UUID{Bytes: orderUUID, Valid: true}, nil
}

func botIDToPgType(botID int64) pgtype.Int8 {
	if botID == 0 {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: botID, Valid: true}
}

func (p *PostgresDB) InsertEvent(ctx context.Context, eventID string, timestamp time.Time, eventType streamtypes.EventType, payload streamtypes.EventPayload) error {
	switch eventType {
	case streamtypes.OrderPlaced:
		ev, ok := payload.(*streamtypes.OrderPlacedEvent)
		if !ok {
			return errors.New("invalid payload type for OrderPlaced event")
		}
		orderUUID, err := orderIDToUUID(ev.OrderID)
		if err != nil {
			return err
		}
		orderPlacedParams := db.HandleOrderPlacedParams{
			ID:              orderUUID,
			UserID:          userIDToPgType(ev.UserID),
			BotID:           botIDToPgType(ev.BotID),
			StockTicker:     ev.StockTicker,
			OrderType:       orderTypeToString(ev.OrderType),
			Side:            orderSideToString(ev.OrderSide),
			Quantity:        ev.Quantity,
			LimitPriceCents: pgtype.Int8{Int64: ev.LimitPriceCents, Valid: true},
		}

		if err = p.db.HandleOrderPlaced(ctx, orderPlacedParams); err != nil {
			return fmt.Errorf("failed to handle order placed: %w", err)
		}
		return nil

	case streamtypes.OrderCancelled:
		ev, ok := payload.(*streamtypes.OrderCancelledEvent)
		if !ok {
			return errors.New("invalid payload type for OrderCancelled event")
		}
		orderUUID, err := orderIDToUUID(ev.OrderID)
		if err != nil {
			return err
		}
		if err = p.db.HandleOrderCancelled(ctx, orderUUID); err != nil {
			return fmt.Errorf("failed to handle order cancelled: %w", err)
		}
		return nil

	case streamtypes.OrderFilled:
		ev, ok := payload.(*streamtypes.OrderFilledEvent)
		if !ok {
			return errors.New("invalid payload type for OrderFilled event")
		}
		orderUUID, err := orderIDToUUID(ev.OrderID)
		if err != nil {
			return err
		}
		if err = p.db.HandleOrderFilled(ctx, orderUUID); err != nil {
			return fmt.Errorf("failed to handle order filled: %w", err)
		}
		return nil

	case streamtypes.OrderPartiallyFilled:
		ev, ok := payload.(*streamtypes.OrderPartiallyFilledEvent)
		if !ok {
			return errors.New("invalid payload type for OrderPartiallyFilled event")
		}
		orderUUID, err := orderIDToUUID(ev.OrderID)
		if err != nil {
			return err
		}
		params := db.HandleOrderPartiallyFilledParams{
			ID:             orderUUID,
			FilledQuantity: pgtype.Int8{Int64: ev.FilledQuantity, Valid: true},
		}
		if err = p.db.HandleOrderPartiallyFilled(ctx, params); err != nil {
			return fmt.Errorf("failed to handle order partially filled: %w", err)
		}
		return nil

	case streamtypes.OrderRejected:
		ev, ok := payload.(*streamtypes.OrderRejectedEvent)
		if !ok {
			return errors.New("invalid payload type for OrderRejected event")
		}
		orderUUID, err := orderIDToUUID(ev.OrderID)
		if err != nil {
			return err
		}
		params := db.HandleOrderRejectedParams{
			ID:     orderUUID,
			UserID: userIDToPgType(ev.UserID),
			BotID:  botIDToPgType(ev.BotID),
		}
		if err = p.db.HandleOrderRejected(ctx, params); err != nil {
			return fmt.Errorf("failed to handle order rejected: %w", err)
		}
		return nil

	case streamtypes.TradeExecuted:
		ev, ok := payload.(*streamtypes.TradeExecutedEvent)
		if !ok {
			return errors.New("invalid payload type for TradeExecuted event")
		}
		tradeUUID, err := orderIDToUUID(ev.TradeID)
		if err != nil {
			return err
		}
		buyerOrderUUID, err := orderIDToUUID(ev.BuyerOrderID)
		if err != nil {
			return err
		}
		sellerOrderUUID, err := orderIDToUUID(ev.SellerOrderID)
		if err != nil {
			return err
		}

		params := db.HandleTradeExecutedParams{
			ID:              tradeUUID,
			StockTicker:     ev.StockTicker,
			BuyerOrderID:    buyerOrderUUID,
			SellerOrderID:   sellerOrderUUID,
			BuyerUserID:     userIDToPgType(ev.BuyerUserID),
			BuyerBotID:      botIDToPgType(ev.BuyerBotID),
			SellerUserID:    userIDToPgType(ev.SellerUserID),
			SellerBotID:     botIDToPgType(ev.SellerBotID),
			Quantity:        ev.Quantity,
			PriceCents:      ev.PriceCents,
			TotalValueCents: ev.TotalValueCents,
		}
		if err = p.db.HandleTradeExecuted(ctx, params); err != nil {
			return fmt.Errorf("failed to handle trade executed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("unsupported event type: %d", eventType)
}
