package matchingengine

import (
	"errors"
	"sync"
	"time"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
)

// MatchingEngine handles order matching for all stocks
type MatchingEngine struct {
	orderBooks sync.Map // stock symbol -> *types.StockOrderBook
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{}
}

// getOrCreateOrderBook gets or creates an order book for a stock
func (me *MatchingEngine) getOrCreateOrderBook(stock string) *types.StockOrderBook {
	// Try to load existing book
	if book, ok := me.orderBooks.Load(stock); ok {
		return book.(*types.StockOrderBook)
	}

	// Create new book and try to store it
	newBook := types.NewStockOrderBook(stock)
	actual, _ := me.orderBooks.LoadOrStore(stock, newBook)
	return actual.(*types.StockOrderBook)
}

// SubmitOrder submits an order and attempts to match it
// Returns a slice of matched events, any remaining unmatched quantity, and an error
func (me *MatchingEngine) SubmitOrder(order *types.Order) ([]types.MatchedEvent, int, error) {
	// Minimal defensive checks to prevent panics
	if order == nil {
		return nil, 0, errors.New("order cannot be nil")
	}
	if order.Stock == "" {
		return nil, 0, errors.New("stock cannot be empty")
	}
	if order.OrderId == "" {
		return nil, 0, errors.New("order ID cannot be empty")
	}

	orderBook := me.getOrCreateOrderBook(order.Stock)

	// Lock only this stock's order book
	orderBook.Mu.Lock()
	defer orderBook.Mu.Unlock()

	if order.OrderSide == types.Buy {
		matches, remaining := me.matchBuyOrder(orderBook, order)
		return matches, remaining, nil
	}
	matches, remaining := me.matchSellOrder(orderBook, order)
	return matches, remaining, nil
}

// matchBuyOrder matches a buy order against the sell side
func (me *MatchingEngine) matchBuyOrder(book *types.StockOrderBook, buyOrder *types.Order) ([]types.MatchedEvent, int) {
	var matches []types.MatchedEvent
	remainingQty := buyOrder.Quantity

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

		level := book.SellSide.GetBestLevel()
		now := time.Now()
		// Match against orders at this price level
		for !level.IsEmpty() && remainingQty > 0 {
			sellOrder := level.Front()

			// Calculate match quantity
			matchQty := min(remainingQty, sellOrder.Quantity)

			// Create match event
			match := types.MatchedEvent{
				BuyerOrderId:       buyOrder.OrderId,
				SellerOrderId:      sellOrder.OrderId,
				PricePerStockCents: askPrice,
				Quantity:           matchQty,
				Timestamp:          now,
			}
			matches = append(matches, match)

			// Update quantities
			remainingQty -= matchQty
			sellOrder.Quantity -= matchQty

			// Remove fully filled sell order
			if sellOrder.Quantity == 0 {
				book.SellSide.RemoveOrder(sellOrder.OrderId)
			}
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
func (me *MatchingEngine) matchSellOrder(book *types.StockOrderBook, sellOrder *types.Order) ([]types.MatchedEvent, int) {
	var matches []types.MatchedEvent
	remainingQty := sellOrder.Quantity

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

			// Update quantities
			remainingQty -= matchQty
			buyOrder.Quantity -= matchQty

			// Remove fully filled buy order
			if buyOrder.Quantity == 0 {
				book.BuySide.RemoveOrder(buyOrder.OrderId)
			}
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

	book := value.(*types.StockOrderBook)
	book.Mu.Lock()
	defer book.Mu.Unlock()

	if side == types.Buy {
		return book.BuySide.RemoveOrder(orderId), nil
	}
	return book.SellSide.RemoveOrder(orderId), nil
}
