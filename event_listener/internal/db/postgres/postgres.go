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

func orderIDToUUID(orderID string) (pgtype.UUID, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("could not parse order UUID: %w", err)
	}
	return pgtype.UUID{Bytes: orderUUID, Valid: true}, nil
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

		// Route to appropriate handler based on order type and side
		if ev.OrderType == streamtypes.LimitOrder && ev.OrderSide == streamtypes.Buy {
			params := db.HandleLimitBuyOrderPlacedParams{
				ID:              orderUUID,
				TraderID:        ev.TraderID,
				StockTicker:     ev.StockTicker,
				Quantity:        ev.Quantity,
				LimitPriceCents: pgtype.Int8{Int64: ev.LimitPriceCents, Valid: true},
			}
			if err = p.db.HandleLimitBuyOrderPlaced(ctx, params); err != nil {
				return fmt.Errorf("failed to handle limit buy order placed: %w", err)
			}
		} else if ev.OrderType == streamtypes.MarketOrder && ev.OrderSide == streamtypes.Buy {
			params := db.HandleMarketBuyOrderPlacedParams{
				ID:          orderUUID,
				TraderID:    ev.TraderID,
				StockTicker: ev.StockTicker,
				Quantity:    ev.Quantity,
			}
			if err = p.db.HandleMarketBuyOrderPlaced(ctx, params); err != nil {
				return fmt.Errorf("failed to handle market buy order placed: %w", err)
			}
		} else if ev.OrderSide == streamtypes.Sell {
			params := db.HandleSellOrderPlacedParams{
				ID:              orderUUID,
				TraderID:        ev.TraderID,
				StockTicker:     ev.StockTicker,
				OrderType:       orderTypeToString(ev.OrderType),
				Quantity:        ev.Quantity,
				LimitPriceCents: pgtype.Int8{Int64: ev.LimitPriceCents, Valid: ev.OrderType == streamtypes.LimitOrder},
			}
			if err = p.db.HandleSellOrderPlaced(ctx, params); err != nil {
				return fmt.Errorf("failed to handle sell order placed: %w", err)
			}
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

		// Route to appropriate handler based on order type and side
		if ev.OrderType == streamtypes.LimitOrder && ev.OrderSide == streamtypes.Buy {
			if err = p.db.HandleLimitBuyOrderCancelled(ctx, orderUUID); err != nil {
				return fmt.Errorf("failed to handle limit buy order cancelled: %w", err)
			}
		} else if ev.OrderType == streamtypes.MarketOrder && ev.OrderSide == streamtypes.Buy {
			if err = p.db.HandleMarketBuyOrderCancelled(ctx, orderUUID); err != nil {
				return fmt.Errorf("failed to handle market buy order cancelled: %w", err)
			}
		} else if ev.OrderSide == streamtypes.Sell {
			if err = p.db.HandleSellOrderCancelled(ctx, orderUUID); err != nil {
				return fmt.Errorf("failed to handle sell order cancelled: %w", err)
			}
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
			ID:       orderUUID,
			TraderID: ev.TraderID,
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
		buyerOrderUUID, err := orderIDToUUID(ev.BuyerOrderID)
		if err != nil {
			return err
		}
		sellerOrderUUID, err := orderIDToUUID(ev.SellerOrderID)
		if err != nil {
			return err
		}

		params := db.HandleLimitBuyTradeExecutedParams{
			StockTicker:     ev.StockTicker,
			BuyerOrderID:    buyerOrderUUID,
			SellerOrderID:   sellerOrderUUID,
			BuyerTraderID:   ev.BuyerTraderID,
			SellerTraderID:  ev.SellerTraderID,
			Quantity:        ev.Quantity,
			PriceCents:      ev.PriceCents,
			TotalValueCents: ev.TotalValueCents,
		}

		// Route to appropriate handler based on buyer's order type
		if ev.BuyerOrderType == streamtypes.LimitOrder {
			if err = p.db.HandleLimitBuyTradeExecuted(ctx, params); err != nil {
				return fmt.Errorf("failed to handle limit buy trade executed: %w", err)
			}
		} else {
			// MarketOrder - need to convert params type
			marketParams := db.HandleMarketBuyTradeExecutedParams{
				StockTicker:     params.StockTicker,
				BuyerOrderID:    params.BuyerOrderID,
				SellerOrderID:   params.SellerOrderID,
				BuyerTraderID:   params.BuyerTraderID,
				SellerTraderID:  params.SellerTraderID,
				Quantity:        params.Quantity,
				PriceCents:      params.PriceCents,
				TotalValueCents: params.TotalValueCents,
			}
			if err = p.db.HandleMarketBuyTradeExecuted(ctx, marketParams); err != nil {
				return fmt.Errorf("failed to handle market buy trade executed: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("unsupported event type: %d", eventType)
}
