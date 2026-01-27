package matchingengine

import (
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
// Returns a slice of matched events and any remaining unmatched quantity
func (me *MatchingEngine) SubmitOrder(order *Order) ([]MatchedEvent, int) {
	orderBook := me.getOrCreateOrderBook(order.stock)

	// Lock only this stock's order book
	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	if order.orderSide == Buy {
		return me.matchBuyOrder(orderBook, order)
	}
	return me.matchSellOrder(orderBook, order)
}

// matchBuyOrder matches a buy order against the sell side
func (me *MatchingEngine) matchBuyOrder(book *StockOrderBook, buyOrder *Order) ([]MatchedEvent, int) {
	var matches []MatchedEvent
	remainingQty := buyOrder.quantity

	// Get sorted sell prices (lowest first - best asks)
	for _, askPrice := range book.sellSide.GetSortedPrices() {
		// For limit orders, check if prices cross
		if buyOrder.orderType == LimitOrder && askPrice > buyOrder.limitPrice {
			break // No more matches possible - remaining sell prices are higher
		}

		level := book.sellSide.levels[askPrice]

		// Match against orders at this price level (FIFO)
		for !level.IsEmpty() && remainingQty > 0 {
			sellOrder := level.Front()

			// Calculate match quantity
			matchQty := min(remainingQty, sellOrder.quantity)

			// Create match event (price is the resting order's price)
			match := MatchedEvent{
				buyerOrderId:       buyOrder.orderId,
				sellerOrderId:      sellOrder.orderId,
				pricePerStockCents: askPrice,
				quantity:           matchQty,
				timestamp:          time.Now(),
			}
			matches = append(matches, match)

			// Update quantities
			remainingQty -= matchQty
			sellOrder.quantity -= matchQty

			// Remove fully filled sell order
			if sellOrder.quantity == 0 {
				book.sellSide.RemoveOrder(sellOrder.orderId)
			} else {
				// Update volume tracking
				level.volume -= matchQty
			}
		}

		if remainingQty == 0 {
			break
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && buyOrder.orderType == LimitOrder {
		buyOrder.quantity = remainingQty
		book.buySide.AddOrder(buyOrder)
	}

	return matches, remainingQty
}

// matchSellOrder matches a sell order against the buy side
func (me *MatchingEngine) matchSellOrder(book *StockOrderBook, sellOrder *Order) ([]MatchedEvent, int) {
	var matches []MatchedEvent
	remainingQty := sellOrder.quantity

	// Get sorted buy prices (highest first - best bids)
	for _, bidPrice := range book.buySide.GetSortedPrices() {
		// For limit orders, check if prices cross
		if sellOrder.orderType == LimitOrder && bidPrice < sellOrder.limitPrice {
			break // No more matches possible - remaining buy prices are lower
		}

		level := book.buySide.levels[bidPrice]

		// Match against orders at this price level (FIFO)
		for !level.IsEmpty() && remainingQty > 0 {
			buyOrder := level.Front()

			// Calculate match quantity
			matchQty := min(remainingQty, buyOrder.quantity)

			// Create match event (price is the resting order's price)
			match := MatchedEvent{
				buyerOrderId:       buyOrder.orderId,
				sellerOrderId:      sellOrder.orderId,
				pricePerStockCents: bidPrice,
				quantity:           matchQty,
				timestamp:          time.Now(),
			}
			matches = append(matches, match)

			// Update quantities
			remainingQty -= matchQty
			buyOrder.quantity -= matchQty

			// Remove fully filled buy order
			if buyOrder.quantity == 0 {
				book.buySide.RemoveOrder(buyOrder.orderId)
			} else {
				// Update volume tracking
				level.volume -= matchQty
			}
		}

		if remainingQty == 0 {
			break
		}
	}

	// If there's remaining quantity for a limit order, add to book
	if remainingQty > 0 && sellOrder.orderType == LimitOrder {
		sellOrder.quantity = remainingQty
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

// GetBestBid returns the best bid price for a stock
func (me *MatchingEngine) GetBestBid(stock string) (int, bool) {
	value, exists := me.orderBooks.Load(stock)
	if !exists {
		return 0, false
	}

	book := value.(*StockOrderBook)
	book.mu.RLock()
	defer book.mu.RUnlock()
	return book.buySide.GetBestPrice()
}

// GetBestAsk returns the best ask price for a stock
func (me *MatchingEngine) GetBestAsk(stock string) (int, bool) {
	value, exists := me.orderBooks.Load(stock)
	if !exists {
		return 0, false
	}

	book := value.(*StockOrderBook)
	book.mu.RLock()
	defer book.mu.RUnlock()
	return book.sellSide.GetBestPrice()
}

// GetSpread returns the bid-ask spread for a stock
func (me *MatchingEngine) GetSpread(stock string) (int, bool) {
	value, exists := me.orderBooks.Load(stock)
	if !exists {
		return 0, false
	}

	book := value.(*StockOrderBook)
	book.mu.RLock()
	defer book.mu.RUnlock()

	bid, hasBid := book.buySide.GetBestPrice()
	ask, hasAsk := book.sellSide.GetBestPrice()

	if !hasBid || !hasAsk {
		return 0, false
	}

	return ask - bid, true
}
