package matchingengine

import (
	"errors"
	"sync"
	"time"
)

// MatchingEngine handles order matching for all stocks
type MatchingEngine struct {
	orderBooks sync.Map // stock symbol -> *StockOrderBook
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{}
}

// getOrCreateOrderBook gets or creates an order book for a stock
func (me *MatchingEngine) getOrCreateOrderBook(stock string) *StockOrderBook {
	// Try to load existing book
	if book, ok := me.orderBooks.Load(stock); ok {
		return book.(*StockOrderBook)
	}

	// Create new book and try to store it
	newBook := NewStockOrderBook(stock)
	actual, _ := me.orderBooks.LoadOrStore(stock, newBook)
	return actual.(*StockOrderBook)
}

// SubmitOrder submits an order and attempts to match it
// Returns a slice of matched events, any remaining unmatched quantity, and an error
func (me *MatchingEngine) SubmitOrder(order *Order) ([]MatchedEvent, int, error) {
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
	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	if order.OrderSide == Buy {
		matches, remaining := me.matchBuyOrder(orderBook, order)
		return matches, remaining, nil
	}
	matches, remaining := me.matchSellOrder(orderBook, order)
	return matches, remaining, nil
}

// matchBuyOrder matches a buy order against the sell side
func (me *MatchingEngine) matchBuyOrder(book *StockOrderBook, buyOrder *Order) ([]MatchedEvent, int) {
	var matches []MatchedEvent
	remainingQty := buyOrder.Quantity

	// Iterate through best asks using heap - O(k log n) where k = matched levels
	for remainingQty > 0 {
		askPrice, ok := book.sellSide.GetBestPrice()
		if !ok {
			break // No more sell orders
		}

		// For limit orders, check if prices cross
		if buyOrder.OrderType == LimitOrder && askPrice > buyOrder.LimitPrice {
			break // No more matches possible - best ask is too high
		}

		level := book.sellSide.levels[askPrice]
		now := time.Now()
		// Match against orders at this price level (FIFO)
		for !level.IsEmpty() && remainingQty > 0 {
			sellOrder := level.Front()

			// Calculate match quantity
			matchQty := min(remainingQty, sellOrder.Quantity)

			// Create match event (price is the resting order's price)
			match := MatchedEvent{
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
				book.sellSide.RemoveOrder(sellOrder.OrderId)
			} else {
				// Update volume tracking
				level.volume -= matchQty
			}
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && buyOrder.OrderType == LimitOrder {
		buyOrder.Quantity = remainingQty
		book.buySide.AddOrder(buyOrder)
	}

	return matches, remainingQty
}

// matchSellOrder matches a sell order against the buy side
func (me *MatchingEngine) matchSellOrder(book *StockOrderBook, sellOrder *Order) ([]MatchedEvent, int) {
	var matches []MatchedEvent
	remainingQty := sellOrder.Quantity

	// Iterate through best bids using heap - O(k log n) where k = matched levels
	for remainingQty > 0 {
		bidPrice, ok := book.buySide.GetBestPrice()
		if !ok {
			break // No more buy orders
		}

		// For limit orders, check if prices cross
		if sellOrder.OrderType == LimitOrder && bidPrice < sellOrder.LimitPrice {
			break // No more matches possible - best bid is too low
		}

		level := book.buySide.levels[bidPrice]
		now := time.Now()
		// Match against orders at this price level (FIFO)
		for !level.IsEmpty() && remainingQty > 0 {
			buyOrder := level.Front()

			// Calculate match quantity
			matchQty := min(remainingQty, buyOrder.Quantity)

			// Create match event (price is the resting order's price)
			match := MatchedEvent{
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
				book.buySide.RemoveOrder(buyOrder.OrderId)
			} else {
				// Update volume tracking
				level.volume -= matchQty
			}
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && sellOrder.OrderType == LimitOrder {
		sellOrder.Quantity = remainingQty
		book.sellSide.AddOrder(sellOrder)
	}

	return matches, remainingQty
}

// CancelOrder cancels an existing order
func (me *MatchingEngine) CancelOrder(stock, orderId string, side OrderSide) bool {
	value, exists := me.orderBooks.Load(stock)
	if !exists {
		return false
	}

	book := value.(*StockOrderBook)
	book.mu.Lock()
	defer book.mu.Unlock()

	if side == Buy {
		return book.buySide.RemoveOrder(orderId)
	}
	return book.sellSide.RemoveOrder(orderId)
}
