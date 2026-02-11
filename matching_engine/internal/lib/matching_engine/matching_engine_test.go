package matchingengine

import (
	"testing"
	"time"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events/streaming_client/clients"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
)

// Helper to create an order
func newOrder(id, stock string, side types.OrderSide, orderType types.OrderType, qty, price int64) *types.Order {
	return &types.Order{
		OrderId:     id,
		StockTicker: stock,
		OrderSide:   side,
		OrderType:   orderType,
		Quantity:    qty,
		LimitPrice:  price,
		Timestamp:   time.Now(),
	}
}

// Helper to create a market buy order with available balance
func newMarketBuyOrder(id, stock string, qty, availableBalance int64) *types.Order {
	return &types.Order{
		OrderId:          id,
		StockTicker:      stock,
		OrderSide:        types.Buy,
		OrderType:        types.MarketOrder,
		Quantity:         qty,
		LimitPrice:       0,
		AvailableBalance: availableBalance,
		Timestamp:        time.Now(),
	}
}

func TestMatchingEngine(t *testing.T) {
	t.Run("should initialize correctly", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)
		if engine == nil {
			t.Fatal("expected engine to be initialized")
		}
	})

	t.Run("should add unmatched limit order to book", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Submit a buy order with no matching sell orders
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 100, 15000) // $150.00
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
		if remaining != 100 {
			t.Errorf("expected 100 remaining, got %d", remaining)
		}
	})

	t.Run("should match crossing orders exactly", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add a sell order at $150
		sellOrder := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 100, 15000)
		engine.SubmitOrder(sellOrder)

		// Add a buy order at $150 that should match
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 100, 15000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}

		match := matches[0]
		if match.BuyerOrderId != "buy1" {
			t.Errorf("expected buyerOrderId 'buy1', got '%s'", match.BuyerOrderId)
		}
		if match.SellerOrderId != "sell1" {
			t.Errorf("expected sellerOrderId 'sell1', got '%s'", match.SellerOrderId)
		}
		if match.Quantity != 100 {
			t.Errorf("expected quantity 100, got %d", match.Quantity)
		}
		if match.PricePerStockCents != 15000 {
			t.Errorf("expected price 15000, got %d", match.PricePerStockCents)
		}
	})

	t.Run("should match at resting order price", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Sell order resting at $145
		sellOrder := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 50, 14500)
		engine.SubmitOrder(sellOrder)

		// Buy order comes in at $150 - should match at $145 (resting price)
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 50, 15000)
		matches, _, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].PricePerStockCents != 14500 {
			t.Errorf("expected match at resting price 14500, got %d", matches[0].PricePerStockCents)
		}
	})

	t.Run("should handle partial fills", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Sell 50 shares at $150
		sellOrder := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 50, 15000)
		engine.SubmitOrder(sellOrder)

		// Buy 100 shares at $150 - should partially fill
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 100, 15000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].Quantity != 50 {
			t.Errorf("expected match quantity 50, got %d", matches[0].Quantity)
		}
		if remaining != 50 {
			t.Errorf("expected 50 remaining, got %d", remaining)
		}
	})

	t.Run("should match multiple orders at same price (FIFO)", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add two sell orders at same price
		sell1 := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 30, 15000)
		sell2 := newOrder("sell2", "AAPL", types.Sell, types.LimitOrder, 30, 15000)
		engine.SubmitOrder(sell1)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		engine.SubmitOrder(sell2)

		// Buy order should match FIFO (sell1 first)
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 50, 15000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if matches[0].SellerOrderId != "sell1" {
			t.Errorf("expected first match with sell1, got %s", matches[0].SellerOrderId)
		}
		if matches[0].Quantity != 30 {
			t.Errorf("expected first match quantity 30, got %d", matches[0].Quantity)
		}
		if matches[1].SellerOrderId != "sell2" {
			t.Errorf("expected second match with sell2, got %s", matches[1].SellerOrderId)
		}
		if matches[1].Quantity != 20 {
			t.Errorf("expected second match quantity 20, got %d", matches[1].Quantity)
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}
	})

	t.Run("should match best price first (price priority)", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add sell orders at different prices
		sell1 := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 50, 15100) // $151.00
		sell2 := newOrder("sell2", "AAPL", types.Sell, types.LimitOrder, 50, 15000) // $150.00 - best ask
		engine.SubmitOrder(sell1)
		engine.SubmitOrder(sell2)

		// Buy should match cheapest first
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 50, 15100)
		matches, _, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].SellerOrderId != "sell2" {
			t.Errorf("expected match with sell2 (best ask), got %s", matches[0].SellerOrderId)
		}
		if matches[0].PricePerStockCents != 15000 {
			t.Errorf("expected price 15000, got %d", matches[0].PricePerStockCents)
		}
	})

	t.Run("should not match if prices don't cross", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Sell at $155
		sellOrder := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 100, 15500)
		engine.SubmitOrder(sellOrder)

		// Buy at $150 - no match
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 100, 15000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
		if remaining != 100 {
			t.Errorf("expected 100 remaining, got %d", remaining)
		}
	})

	t.Run("should cancel orders", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add a buy order
		buyOrder := newOrder("buy1", "AAPL", types.Buy, types.LimitOrder, 100, 15000)
		engine.SubmitOrder(buyOrder)

		// Cancel it
		cancelled, err := engine.CancelOrder("AAPL", "buy1", types.Buy)
		if err != nil {
			t.Errorf("unexpected error : %s", err.Error())
		}
		if !cancelled {
			t.Error("expected order to be cancelled")
		}

		// Try to cancel again - should return false
		cancelledAgain, err := engine.CancelOrder("AAPL", "buy1", types.Buy)
		if err != nil {
			t.Errorf("unexpected error : %s", err.Error())
		}
		if cancelledAgain {
			t.Error("expected cancel to return false for already cancelled order")
		}
	})

	t.Run("should handle market orders", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add sell orders at different prices
		sell1 := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 50, 15000)
		sell2 := newOrder("sell2", "AAPL", types.Sell, types.LimitOrder, 50, 15100)
		engine.SubmitOrder(sell1)
		engine.SubmitOrder(sell2)

		// Market buy should match all available (has enough balance: 50*15000 + 50*15100 = 1505000)
		buyOrder := newMarketBuyOrder("buy1", "AAPL", 100, 1_600_000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining, got %d", remaining)
		}
	})

	t.Run("should stop market buy when balance exhausted", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add sell orders: 50 @ $150.00 and 50 @ $151.00
		sell1 := newOrder("sell1", "AAPL", types.Sell, types.LimitOrder, 50, 15000)
		sell2 := newOrder("sell2", "AAPL", types.Sell, types.LimitOrder, 50, 15100)
		engine.SubmitOrder(sell1)
		engine.SubmitOrder(sell2)

		// Buyer wants 100 shares but only has $800,000 (can afford 50@15000 + 3@15100 = 795300)
		buyOrder := newMarketBuyOrder("buy1", "AAPL", 100, 800_000)
		matches, remaining, _ := engine.SubmitOrder(buyOrder)

		// Should fill 50 @ 15000 (cost=750000, remaining balance=50000)
		// Then 3 @ 15100 (cost=45300, remaining balance=4700 < 15100, stop)
		// Remaining qty = 100 - 50 - 3 = 47, but IOC cancels remainder → remaining = 0
		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if matches[0].Quantity != 50 {
			t.Errorf("expected first match qty 50, got %d", matches[0].Quantity)
		}
		if matches[1].Quantity != 3 {
			t.Errorf("expected second match qty 3, got %d", matches[1].Quantity)
		}
		// Unfilled portion = 47 (IOC cancellation event is emitted separately)
		if remaining != 47 {
			t.Errorf("expected 47 remaining, got %d", remaining)
		}
	})

	t.Run("should handle multiple stocks independently", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		// Add orders for different stocks
		aaplSell := newOrder("aapl-sell", "AAPL", types.Sell, types.LimitOrder, 100, 15000)
		googlSell := newOrder("googl-sell", "GOOGL", types.Sell, types.LimitOrder, 50, 140000)
		engine.SubmitOrder(aaplSell)
		engine.SubmitOrder(googlSell)

		// Buy AAPL only
		aaplBuy := newOrder("aapl-buy", "AAPL", types.Buy, types.LimitOrder, 100, 15000)
		matches, _, _ := engine.SubmitOrder(aaplBuy)

		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0].SellerOrderId != "aapl-sell" {
			t.Errorf("expected match with aapl-sell, got %s", matches[0].SellerOrderId)
		}
	})
}

func TestMatchingEngineConcurrency(t *testing.T) {
	t.Run("should handle concurrent orders for same stock", func(t *testing.T) {
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)

		done := make(chan bool)

		// Submit many orders concurrently
		for i := range 100 {
			go func(id int) {
				order := newOrder(
					"order-"+string(rune('A'+id%26))+string(rune('0'+id%10)),
					"AAPL",
					types.OrderSide(id%2),
					types.LimitOrder,
					10,
					int64(15000+id),
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
		testStreamingClient := &clients.TestStreamingClient{}
		engine := NewMatchingEngine(testStreamingClient)
		stocks := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "META"}

		done := make(chan bool)

		// Submit orders for different stocks concurrently
		for i := range 100 {
			go func(id int) {
				order := newOrder(
					"order-"+string(rune('A'+id)),
					stocks[id%len(stocks)],
					types.OrderSide(id%2),
					types.LimitOrder,
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
