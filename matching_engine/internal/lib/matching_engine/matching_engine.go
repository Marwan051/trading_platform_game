package matchingengine

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	streamingclient "github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events/streaming_client"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
)

// MatchingEngine handles order matching for all stocks
type MatchingEngine struct {
	orderBooks    sync.Map // stock symbol -> *types.StockOrderBook
	eventStreamer streamingclient.StreamingClient
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(streamer streamingclient.StreamingClient) *MatchingEngine {
	return &MatchingEngine{eventStreamer: streamer}
}

func (me *MatchingEngine) IsEventStreamerHealthy(ctx context.Context) (bool, error) {
	ok, err := me.eventStreamer.IsHealthy(ctx)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// safePublish sends events with a bounded timeout and logs failures.
// Keeps matching logic from blocking indefinitely on I/O.
func (me *MatchingEngine) safePublish(evt any, et types.EventType) {
	if me.eventStreamer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := me.eventStreamer.Publish(ctx, evt, et); err != nil {
		log.Printf("event publish failed: %v, type=%d", err, int(et))
	}
}

// getOrCreateOrderBook gets or creates an order book for a stock
func (me *MatchingEngine) getOrCreateOrderBook(stock string) *types.StockOrderBook {
	// Try to load existing book
	if book, ok := me.orderBooks.Load(stock); ok {
		if orderBook, ok := book.(*types.StockOrderBook); ok {
			return orderBook
		}
		// Fallback: corrupt data, create new book
	}

	// Create new book and try to store it
	newBook := types.NewStockOrderBook(stock)
	actual, _ := me.orderBooks.LoadOrStore(stock, newBook)
	if orderBook, ok := actual.(*types.StockOrderBook); ok {
		return orderBook
	}
	// Should never happen, but return new book as fallback
	return newBook
}

// SubmitOrder submits an order and attempts to match it
// Returns a slice of matched events, any remaining unmatched quantity, and an error
func (me *MatchingEngine) SubmitOrder(order *types.Order) ([]types.MatchedEvent, int64, error) {
	// Minimal defensive checks to prevent panics
	if order == nil {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderRejectedEvent{
				OrderID:      "",
				TraderID:     0,
				Reason:       "Order is empty",
				ErrorMessage: "Order is empty",
			}, types.OrderRejected)
		}
		return nil, 0, errors.New("order cannot be nil")
	}
	if order.StockTicker == "" {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderRejectedEvent{
				OrderID:      "",
				TraderID:     0,
				Reason:       "Ticker is empty",
				ErrorMessage: "Ticker is empty",
			}, types.OrderRejected)
		}
		return nil, 0, errors.New("stock cannot be empty")
	}
	if order.OrderId == "" {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderRejectedEvent{
				OrderID:      "",
				TraderID:     0,
				Reason:       "OrderId is empty",
				ErrorMessage: "OrderId is empty",
			}, types.OrderRejected)
		}
		return nil, 0, errors.New("order ID cannot be empty")
	}
	if order.Quantity <= 0 {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderRejectedEvent{
				OrderID:      order.OrderId,
				TraderID:     order.TraderId,
				Reason:       "Invalid quantity",
				ErrorMessage: "Quantity must be greater than 0",
			}, types.OrderRejected)
		}
		return nil, 0, errors.New("quantity must be greater than 0")
	}
	if order.OrderType == types.LimitOrder && order.LimitPrice <= 0 {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderRejectedEvent{
				OrderID:      order.OrderId,
				TraderID:     order.TraderId,
				Reason:       "Invalid limit price",
				ErrorMessage: "Limit price must be greater than 0",
			}, types.OrderRejected)
		}
		return nil, 0, errors.New("limit price must be greater than 0")
	}

	orderBook := me.getOrCreateOrderBook(order.StockTicker)

	// Lock only this stock's order book
	orderBook.Mu.Lock()
	defer orderBook.Mu.Unlock()

	// Emit OrderPlacedEvent - order has been accepted
	if me.eventStreamer != nil {
		me.safePublish(&types.OrderPlacedEvent{
			OrderID:         order.OrderId,
			TraderID:        order.TraderId,
			StockTicker:     order.StockTicker,
			OrderType:       order.OrderType,
			OrderSide:       order.OrderSide,
			Quantity:        order.Quantity,
			LimitPriceCents: order.LimitPrice,
		}, types.OrderPlaced)
	}

	if order.OrderSide == types.Buy {
		matches, remaining := me.matchBuyOrder(orderBook, order)
		return matches, remaining, nil
	}
	matches, remaining := me.matchSellOrder(orderBook, order)
	return matches, remaining, nil
}

// matchBuyOrder matches a buy order against the sell side
func (me *MatchingEngine) matchBuyOrder(book *types.StockOrderBook, buyOrder *types.Order) ([]types.MatchedEvent, int64) {
	var matches []types.MatchedEvent
	remainingQty := buyOrder.Quantity
	originalBuyQty := buyOrder.Quantity
	remainingBalance := buyOrder.AvailableBalance // Track remaining cash for market orders

	// Iterate through best asks using heap
	for remainingQty > 0 {
		askPrice, ok := book.SellSide.GetBestPrice()
		if !ok {
			break // No more sell orders
		}

		// For limit orders, check if prices cross
		if buyOrder.OrderType == types.LimitOrder && askPrice > buyOrder.LimitPrice {
			break // No more matches possible
		}

		// For market orders, check if buyer can afford at least 1 share at this price
		if buyOrder.OrderType == types.MarketOrder && remainingBalance < askPrice {
			break // Buyer can't afford any more shares
		}

		level := book.SellSide.GetBestLevel()
		now := time.Now()
		// Match against orders at this price level
		for !level.IsEmpty() && remainingQty > 0 {
			// For market orders, re-check affordability at this price level
			if buyOrder.OrderType == types.MarketOrder && remainingBalance < askPrice {
				break
			}

			sellOrder := level.Front()
			originalSellQty := sellOrder.Quantity

			// Calculate match quantity
			matchQty := min(remainingQty, sellOrder.Quantity)

			// For market orders, cap quantity by what the buyer can actually afford
			if buyOrder.OrderType == types.MarketOrder {
				affordableQty := remainingBalance / askPrice
				if affordableQty < matchQty {
					matchQty = affordableQty
				}
				if matchQty == 0 {
					remainingQty = 0 // Force exit — can't afford any more
					break
				}
			}

			tradeCost := askPrice * matchQty

			// Create match event
			match := types.MatchedEvent{
				BuyerOrderId:       buyOrder.OrderId,
				SellerOrderId:      sellOrder.OrderId,
				PricePerStockCents: askPrice,
				Quantity:           matchQty,
				Timestamp:          now,
			}
			matches = append(matches, match)
			if me.eventStreamer != nil {
				me.safePublish(&types.TradeExecutedEvent{
					StockTicker:     buyOrder.StockTicker,
					BuyerOrderID:    buyOrder.OrderId,
					SellerOrderID:   sellOrder.OrderId,
					BuyerOrderType:  buyOrder.OrderType,
					BuyerTraderID:   buyOrder.TraderId,
					SellerTraderID:  sellOrder.TraderId,
					Quantity:        matchQty,
					PriceCents:      askPrice,
					TotalValueCents: tradeCost,
				}, types.TradeExecuted)
			}
			// Update quantities
			remainingQty -= matchQty
			sellOrder.Quantity -= matchQty

			// Track spend for market orders
			if buyOrder.OrderType == types.MarketOrder {
				remainingBalance -= tradeCost
			}

			// Emit events for the resting sell order
			if sellOrder.Quantity == 0 {
				// Resting sell order fully filled
				if me.eventStreamer != nil {
					me.safePublish(&types.OrderFilledEvent{
						OrderID:        sellOrder.OrderId,
						TraderID:       sellOrder.TraderId,
						Quantity:       originalSellQty,
						FillPriceCents: askPrice,
					}, types.OrderFilled)
				}
				book.SellSide.RemoveOrder(sellOrder.OrderId)
			} else {
				// Resting sell order partially filled
				if me.eventStreamer != nil {
					me.safePublish(&types.OrderPartiallyFilledEvent{
						OrderID:           sellOrder.OrderId,
						TraderID:          sellOrder.TraderId,
						FilledQuantity:    matchQty,
						RemainingQuantity: sellOrder.Quantity,
						FillPriceCents:    askPrice,
					}, types.OrderPartiallyFilled)
				}
			}
		}
	}

	// Emit OrderFilledEvent for incoming buy order if fully consumed
	if remainingQty == 0 && originalBuyQty > 0 {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderFilledEvent{
				OrderID:        buyOrder.OrderId,
				TraderID:       buyOrder.TraderId,
				Quantity:       originalBuyQty,
				FillPriceCents: 0, // Client calculates from partial events
			}, types.OrderFilled)
		}
	} else if remainingQty > 0 && remainingQty < originalBuyQty {
		// Emit single partial event for incoming buy order if partially filled
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderPartiallyFilledEvent{
				OrderID:           buyOrder.OrderId,
				TraderID:          buyOrder.TraderId,
				FilledQuantity:    originalBuyQty - remainingQty,
				RemainingQuantity: remainingQty,
				FillPriceCents:    0, // Multiple fills at different prices
			}, types.OrderPartiallyFilled)
		}
	}

	// Market orders: cancel unfilled portion (IOC behavior)
	if remainingQty > 0 && buyOrder.OrderType == types.MarketOrder {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderCancelledEvent{
				OrderID:           buyOrder.OrderId,
				TraderID:          buyOrder.TraderId,
				OrderType:         buyOrder.OrderType,
				OrderSide:         buyOrder.OrderSide,
				StockTicker:       buyOrder.StockTicker,
				RemainingQuantity: remainingQty,
			}, types.OrderCancelled)
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && buyOrder.OrderType == types.LimitOrder {
		buyOrder.Quantity = remainingQty
		book.BuySide.AddOrder(buyOrder)
	}

	return matches, remainingQty
}

// matchSellOrder matches a sell order against the buy side
func (me *MatchingEngine) matchSellOrder(book *types.StockOrderBook, sellOrder *types.Order) ([]types.MatchedEvent, int64) {
	var matches []types.MatchedEvent
	remainingQty := sellOrder.Quantity
	originalSellQty := sellOrder.Quantity

	// Iterate through best bids using heap
	for remainingQty > 0 {
		bidPrice, ok := book.BuySide.GetBestPrice()
		if !ok {
			break // No more buy orders
		}

		// For limit orders, check if prices cross
		if sellOrder.OrderType == types.LimitOrder && bidPrice < sellOrder.LimitPrice {
			break // No more matches possible
		}

		level := book.BuySide.GetBestLevel()
		now := time.Now()
		// Match against orders at this price level
		for !level.IsEmpty() && remainingQty > 0 {
			buyOrder := level.Front()
			originalBuyQty := buyOrder.Quantity

			// Calculate match quantity
			matchQty := min(remainingQty, buyOrder.Quantity)

			// Create match event
			match := types.MatchedEvent{
				BuyerOrderId:       buyOrder.OrderId,
				SellerOrderId:      sellOrder.OrderId,
				PricePerStockCents: bidPrice,
				Quantity:           matchQty,
				Timestamp:          now,
			}
			matches = append(matches, match)

			// Emit trade executed event
			if me.eventStreamer != nil {
				me.safePublish(&types.TradeExecutedEvent{
					StockTicker:     sellOrder.StockTicker,
					BuyerOrderID:    buyOrder.OrderId,
					SellerOrderID:   sellOrder.OrderId,
					BuyerOrderType:  buyOrder.OrderType,
					BuyerTraderID:   buyOrder.TraderId,
					SellerTraderID:  sellOrder.TraderId,
					Quantity:        matchQty,
					PriceCents:      bidPrice,
					TotalValueCents: bidPrice * matchQty,
				}, types.TradeExecuted)
			}

			// Update quantities
			remainingQty -= matchQty
			buyOrder.Quantity -= matchQty

			// Emit events for the resting buy order
			if buyOrder.Quantity == 0 {
				// Resting buy order fully filled
				if me.eventStreamer != nil {
					me.safePublish(&types.OrderFilledEvent{
						OrderID:        buyOrder.OrderId,
						TraderID:       buyOrder.TraderId,
						Quantity:       originalBuyQty,
						FillPriceCents: bidPrice,
					}, types.OrderFilled)
				}
				book.BuySide.RemoveOrder(buyOrder.OrderId)
			} else {
				// Resting buy order partially filled
				if me.eventStreamer != nil {
					me.safePublish(&types.OrderPartiallyFilledEvent{
						OrderID:           buyOrder.OrderId,
						TraderID:          buyOrder.TraderId,
						FilledQuantity:    matchQty,
						RemainingQuantity: buyOrder.Quantity,
						FillPriceCents:    bidPrice,
					}, types.OrderPartiallyFilled)
				}
			}
		}
	}

	// Emit OrderFilledEvent for incoming sell order if fully consumed
	if remainingQty == 0 && originalSellQty > 0 {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderFilledEvent{
				OrderID:        sellOrder.OrderId,
				TraderID:       sellOrder.TraderId,
				Quantity:       originalSellQty,
				FillPriceCents: 0, // Client calculates from partial events
			}, types.OrderFilled)
		}
	} else if remainingQty > 0 && remainingQty < originalSellQty {
		// Emit single partial event for incoming sell order if partially filled
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderPartiallyFilledEvent{
				OrderID:           sellOrder.OrderId,
				TraderID:          sellOrder.TraderId,
				FilledQuantity:    originalSellQty - remainingQty,
				RemainingQuantity: remainingQty,
				FillPriceCents:    0, // Multiple fills at different prices
			}, types.OrderPartiallyFilled)
		}
	}

	// Market orders: cancel unfilled portion (IOC behavior)
	if remainingQty > 0 && sellOrder.OrderType == types.MarketOrder {
		if me.eventStreamer != nil {
			me.safePublish(&types.OrderCancelledEvent{
				OrderID:           sellOrder.OrderId,
				TraderID:          sellOrder.TraderId,
				OrderType:         sellOrder.OrderType,
				OrderSide:         sellOrder.OrderSide,
				StockTicker:       sellOrder.StockTicker,
				RemainingQuantity: remainingQty,
			}, types.OrderCancelled)
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && sellOrder.OrderType == types.LimitOrder {
		sellOrder.Quantity = remainingQty
		book.SellSide.AddOrder(sellOrder)
	}

	return matches, remainingQty
}

// CancelOrder cancels an existing order
// Returns (found, error) where found indicates if the order was found and canceled
func (me *MatchingEngine) CancelOrder(stock, orderId string, side types.OrderSide) (bool, error) {
	// Validate inputs
	if stock == "" {
		return false, errors.New("stock cannot be empty")
	}
	if orderId == "" {
		return false, errors.New("order ID cannot be empty")
	}

	value, exists := me.orderBooks.Load(stock)
	if !exists {
		return false, nil // Order book doesn't exist, so order not found
	}

	book, ok := value.(*types.StockOrderBook)
	if !ok {
		return false, errors.New("invalid order book type in sync.Map")
	}
	book.Mu.Lock()
	defer book.Mu.Unlock()

	var order *types.Order
	var removed bool
	if side == types.Buy {
		order, removed = book.BuySide.RemoveOrder(orderId)
	} else {
		order, removed = book.SellSide.RemoveOrder(orderId)
	}
	if !removed {
		return false, nil
	}

	if me.eventStreamer != nil {
		me.safePublish(&types.OrderCancelledEvent{
			OrderID:           order.OrderId,
			TraderID:          order.TraderId,
			OrderType:         order.OrderType,
			OrderSide:         order.OrderSide,
			StockTicker:       order.StockTicker,
			RemainingQuantity: order.Quantity,
		}, types.OrderCancelled)
	}

	return true, nil
}
