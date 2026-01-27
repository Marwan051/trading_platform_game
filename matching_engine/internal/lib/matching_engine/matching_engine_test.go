package matchingengine

import (
	"testing"
	"time"
)

// Helper to create an order
func newOrder(id, stock string, side OrderSide, orderType OrderType, qty, price int) *Order {
	return &Order{
		orderId:    id,
		stock:      stock,
		orderSide:  side,
		orderType:  orderType,
		quantity:   qty,
		limitPrice: price,
		timestamp:  time.Now(),
	}
}

func TestMatchingEngine(t *testing.T) {
	t.Run("should initialize correctly", func(t *testing.T) {
		engine := NewMatchingEngine()
		if engine == nil {
			t.Fatal("expected engine to be initialized")
		}
	})

	t.Run("should add unmatched limit order to book", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Submit a buy order with no matching sell orders
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 100, 15000) // $150.00
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
		if remaining != 100 {
			t.Errorf("expected 100 remaining, got %d", remaining)
		}

		// Check that best bid is now $150.00
		bestBid, ok := engine.GetBestBid("AAPL")
		if !ok || bestBid != 15000 {
			t.Errorf("expected best bid 15000, got %d (ok=%v)", bestBid, ok)
		}
	})

	t.Run("should match crossing orders exactly", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add a sell order at $150
		sellOrder := newOrder("sell1", "AAPL", Sell, LimitOrder, 100, 15000)
		engine.SubmitOrder(sellOrder)

		// Add a buy order at $150 - should match
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 100, 15000)
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}

		match := matches[0]
		if match.buyerOrderId != "buy1" {
			t.Errorf("expected buyerOrderId 'buy1', got '%s'", match.buyerOrderId)
		}
		if match.sellerOrderId != "sell1" {
			t.Errorf("expected sellerOrderId 'sell1', got '%s'", match.sellerOrderId)
		}
		if match.quantity != 100 {
			t.Errorf("expected quantity 100, got %d", match.quantity)
		}
		if match.pricePerStockCents != 15000 {
			t.Errorf("expected price 15000, got %d", match.pricePerStockCents)
		}
	})

	t.Run("should match at resting order price", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Sell order resting at $145
		sellOrder := newOrder("sell1", "AAPL", Sell, LimitOrder, 50, 14500)
		engine.SubmitOrder(sellOrder)

		// Buy order comes in at $150 - should match at $145 (resting price)
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 50, 15000)
		matches, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].pricePerStockCents != 14500 {
			t.Errorf("expected match at resting price 14500, got %d", matches[0].pricePerStockCents)
		}
	})

	t.Run("should handle partial fills", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Sell 50 shares at $150
		sellOrder := newOrder("sell1", "AAPL", Sell, LimitOrder, 50, 15000)
		engine.SubmitOrder(sellOrder)

		// Buy 100 shares at $150 - should partially fill
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 100, 15000)
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].quantity != 50 {
			t.Errorf("expected match quantity 50, got %d", matches[0].quantity)
		}
		if remaining != 50 {
			t.Errorf("expected 50 remaining, got %d", remaining)
		}

		// Remaining 50 should be resting on buy side
		bestBid, ok := engine.GetBestBid("AAPL")
		if !ok || bestBid != 15000 {
			t.Errorf("expected best bid 15000, got %d", bestBid)
		}
	})

	t.Run("should match multiple orders at same price (FIFO)", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add two sell orders at same price
		sell1 := newOrder("sell1", "AAPL", Sell, LimitOrder, 30, 15000)
		sell2 := newOrder("sell2", "AAPL", Sell, LimitOrder, 30, 15000)
		engine.SubmitOrder(sell1)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		engine.SubmitOrder(sell2)

		// Buy order should match FIFO (sell1 first)
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 50, 15000)
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if matches[0].sellerOrderId != "sell1" {
			t.Errorf("expected first match with sell1, got %s", matches[0].sellerOrderId)
		}
		if matches[0].quantity != 30 {
			t.Errorf("expected first match quantity 30, got %d", matches[0].quantity)
		}
		if matches[1].sellerOrderId != "sell2" {
			t.Errorf("expected second match with sell2, got %s", matches[1].sellerOrderId)
		}
		if matches[1].quantity != 20 {
			t.Errorf("expected second match quantity 20, got %d", matches[1].quantity)
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}
	})

	t.Run("should match best price first (price priority)", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add sell orders at different prices
		sell1 := newOrder("sell1", "AAPL", Sell, LimitOrder, 50, 15100) // $151.00
		sell2 := newOrder("sell2", "AAPL", Sell, LimitOrder, 50, 15000) // $150.00 - best ask
		engine.SubmitOrder(sell1)
		engine.SubmitOrder(sell2)

		// Buy should match cheapest first
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 50, 15100)
		matches, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].sellerOrderId != "sell2" {
			t.Errorf("expected match with sell2 (best ask), got %s", matches[0].sellerOrderId)
		}
		if matches[0].pricePerStockCents != 15000 {
			t.Errorf("expected price 15000, got %d", matches[0].pricePerStockCents)
		}
	})

	t.Run("should not match if prices don't cross", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Sell at $155
		sellOrder := newOrder("sell1", "AAPL", Sell, LimitOrder, 100, 15500)
		engine.SubmitOrder(sellOrder)

		// Buy at $150 - no match
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 100, 15000)
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
		if remaining != 100 {
			t.Errorf("expected 100 remaining, got %d", remaining)
		}

		// Verify spread
		spread, ok := engine.GetSpread("AAPL")
		if !ok {
			t.Fatal("expected spread to exist")
		}
		if spread != 500 { // $5.00 spread
			t.Errorf("expected spread 500, got %d", spread)
		}
	})

	t.Run("should cancel orders", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add a buy order
		buyOrder := newOrder("buy1", "AAPL", Buy, LimitOrder, 100, 15000)
		engine.SubmitOrder(buyOrder)

		// Verify it's in the book
		_, ok := engine.GetBestBid("AAPL")
		if !ok {
			t.Fatal("expected buy order in book")
		}

		// Cancel it
		cancelled := engine.CancelOrder("AAPL", "buy1", Buy)
		if !cancelled {
			t.Error("expected order to be cancelled")
		}

		// Verify it's gone
		_, ok = engine.GetBestBid("AAPL")
		if ok {
			t.Error("expected no best bid after cancellation")
		}
	})

	t.Run("should handle market orders", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add sell orders at different prices
		sell1 := newOrder("sell1", "AAPL", Sell, LimitOrder, 50, 15000)
		sell2 := newOrder("sell2", "AAPL", Sell, LimitOrder, 50, 15100)
		engine.SubmitOrder(sell1)
		engine.SubmitOrder(sell2)

		// Market buy should match all available
		buyOrder := newOrder("buy1", "AAPL", Buy, MarketOrder, 100, 0)
		matches, remaining := engine.SubmitOrder(buyOrder)

		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}
		// Market order should NOT rest in the book
		_, ok := engine.GetBestBid("AAPL")
		if ok {
			t.Error("market order should not rest in book")
		}
	})

	t.Run("should handle multiple stocks independently", func(t *testing.T) {
		engine := NewMatchingEngine()

		// Add orders for different stocks
		aaplSell := newOrder("aapl-sell", "AAPL", Sell, LimitOrder, 100, 15000)
		googlSell := newOrder("googl-sell", "GOOGL", Sell, LimitOrder, 50, 140000)
		engine.SubmitOrder(aaplSell)
		engine.SubmitOrder(googlSell)

		// Buy AAPL only
		aaplBuy := newOrder("aapl-buy", "AAPL", Buy, LimitOrder, 100, 15000)
		matches, _ := engine.SubmitOrder(aaplBuy)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].sellerOrderId != "aapl-sell" {
			t.Errorf("expected match with aapl-sell, got %s", matches[0].sellerOrderId)
		}

		// GOOGL should be unaffected
		googlAsk, ok := engine.GetBestAsk("GOOGL")
		if !ok || googlAsk != 140000 {
			t.Errorf("expected GOOGL best ask 140000, got %d", googlAsk)
		}
	})
}

func TestMatchingEngineConcurrency(t *testing.T) {
	t.Run("should handle concurrent orders for same stock", func(t *testing.T) {
		engine := NewMatchingEngine()

		done := make(chan bool)

		// Submit many orders concurrently
		for i := range 100 {
			go func(id int) {
				order := newOrder(
					"order-"+string(rune('A'+id%26))+string(rune('0'+id%10)),
					"AAPL",
					OrderSide(id%2),
					LimitOrder,
					10,
					15000+id,
				)
				engine.SubmitOrder(order)
				done <- true
			}(i)
		}

		// Wait for all to complete
		for range 100 {
			<-done
		}

		// Should not panic or deadlock - reaching here is success
	})

	t.Run("should handle concurrent orders for different stocks", func(t *testing.T) {
		engine := NewMatchingEngine()
		stocks := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "META"}

		done := make(chan bool)

		// Submit orders for different stocks concurrently
		for i := range 100 {
			go func(id int) {
				order := newOrder(
					"order-"+string(rune('A'+id)),
					stocks[id%len(stocks)],
					OrderSide(id%2),
					LimitOrder,
					10,
					15000,
				)
				engine.SubmitOrder(order)
				done <- true
			}(i)
		}

		for range 100 {
			<-done
		}

		// Should not panic or deadlock
	})
}
