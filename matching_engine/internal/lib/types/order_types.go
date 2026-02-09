package types

import (
	"container/heap"
	"container/list"
	"sync"
	"time"
)

// OrderType - The type of order
type OrderType int

const (
	MarketOrder OrderType = iota
	LimitOrder
)

// OrderSide - On What side of the order the request falls
type OrderSide int

const (
	Buy OrderSide = iota
	Sell
)

// Order - The order itself
type Order struct {
	OrderId     string
	UserId      string
	BotId       string
	StockTicker string
	OrderType   OrderType
	OrderSide   OrderSide
	Quantity    int64
	LimitPrice  int64
	Timestamp   time.Time
}

// MatchedEvent represents a successful trade between buyer and seller
type MatchedEvent struct {
	BuyerOrderId       string
	SellerOrderId      string
	PricePerStockCents int64
	Quantity           int64
	Timestamp          time.Time
}

// PriceLevel represents all orders at a specific price
type PriceLevel struct {
	price  int64
	orders *list.List // Doubly-linked list for FIFO ordering
	volume int64      // Total quantity at this price level
}

// NewPriceLevel creates a new price level
func NewPriceLevel(price int64) *PriceLevel {
	return &PriceLevel{
		price:  price,
		orders: list.New(),
		volume: 0,
	}
}

// AddOrder adds an order to this price level (appends to end for FIFO)
func (pl *PriceLevel) AddOrder(order *Order) *list.Element {
	pl.volume += order.Quantity
	return pl.orders.PushBack(order)
}

// Front returns the first order at this price level
func (pl *PriceLevel) Front() *Order {
	if pl.orders.Len() == 0 {
		return nil
	}
	return pl.orders.Front().Value.(*Order)
}

// RemoveOrder removes an order from this price level
func (pl *PriceLevel) RemoveOrder(element *list.Element) {
	order := element.Value.(*Order)
	pl.volume -= order.Quantity
	pl.orders.Remove(element)
}

// RemoveFront removes and returns the first order
func (pl *PriceLevel) RemoveFront() *Order {
	if pl.orders.Len() == 0 {
		return nil
	}
	element := pl.orders.Front()
	order := element.Value.(*Order)
	pl.volume -= order.Quantity
	pl.orders.Remove(element)
	return order
}

// IsEmpty checks if the price level has no orders
func (pl *PriceLevel) IsEmpty() bool {
	return pl.orders.Len() == 0
}

// PriceHeap implements heap.Interface for price levels
// For buy side: max-heap (highest price first)
// For sell side: min-heap (lowest price first)
type PriceHeap struct {
	prices    []int64
	isBuySide bool
}

func (h PriceHeap) Len() int { return len(h.prices) }

func (h PriceHeap) Less(i, j int) bool {
	if h.isBuySide {
		return h.prices[i] > h.prices[j] // Max-heap for buy side
	}
	return h.prices[i] < h.prices[j] // Min-heap for sell side
}

func (h PriceHeap) Swap(i, j int) { h.prices[i], h.prices[j] = h.prices[j], h.prices[i] }

func (h *PriceHeap) Push(x any) {
	h.prices = append(h.prices, x.(int64))
}

func (h *PriceHeap) Pop() any {
	old := h.prices
	n := len(old)
	x := old[n-1]
	h.prices = old[0 : n-1]
	return x
}

// Peek returns the best price without removing it
func (h *PriceHeap) Peek() (int64, bool) {
	if len(h.prices) == 0 {
		return 0, false
	}
	return h.prices[0], true
}

// OrderBookSide represents one side of the order book (buy or sell)
type OrderBookSide struct {
	levels       map[int64]*PriceLevel    // price -> PriceLevel
	priceHeap    *PriceHeap               // Heap for O(log n) best price access
	orderLookup  map[string]*list.Element // orderId -> list element for O(1) cancellation
	orderToPrice map[string]int64         // orderId -> price for lookup
	isBuySide    bool                     // true for buy side, false for sell side
}

// NewOrderBookSide creates a new order book side
func NewOrderBookSide(isBuySide bool) *OrderBookSide {
	h := &PriceHeap{prices: make([]int64, 0), isBuySide: isBuySide}
	heap.Init(h)
	return &OrderBookSide{
		levels:       make(map[int64]*PriceLevel),
		priceHeap:    h,
		orderLookup:  make(map[string]*list.Element),
		orderToPrice: make(map[string]int64),
		isBuySide:    isBuySide,
	}
}

// AddOrder adds an order to the order book side
func (obs *OrderBookSide) AddOrder(order *Order) {
	price := order.LimitPrice

	// Create price level if it doesn't exist
	if _, exists := obs.levels[price]; !exists {
		obs.levels[price] = NewPriceLevel(price)
		heap.Push(obs.priceHeap, price)
	}

	// Add order to price level
	element := obs.levels[price].AddOrder(order)
	obs.orderLookup[order.OrderId] = element
	obs.orderToPrice[order.OrderId] = price
}

// RemoveOrder removes an order by ID and returns the removed order.
func (obs *OrderBookSide) RemoveOrder(orderId string) (*Order, bool) {
	element, exists := obs.orderLookup[orderId]
	if !exists {
		return nil, false
	}

	order := element.Value.(*Order)
	price := obs.orderToPrice[orderId]
	level := obs.levels[price]

	level.RemoveOrder(element)
	delete(obs.orderLookup, orderId)
	delete(obs.orderToPrice, orderId)

	// Remove empty price level from map (lazy deletion in heap)
	if level.IsEmpty() {
		delete(obs.levels, price)
		// Price remains in heap as stale entry - cleaned up lazily in GetBestPrice/PopBestPrice
	}

	return order, true
}

// cleanStaleHeapTop removes stale prices from heap top (prices with no level)
func (obs *OrderBookSide) cleanStaleHeapTop() {
	for obs.priceHeap.Len() > 0 {
		topPrice, _ := obs.priceHeap.Peek()
		if _, exists := obs.levels[topPrice]; exists {
			break // Top price is valid
		}
		// Remove stale price
		heap.Pop(obs.priceHeap)
	}
}

// GetBestPrice returns the best price on this side (highest for buy, lowest for sell)
// Uses lazy deletion to skip stale prices
func (obs *OrderBookSide) GetBestPrice() (int64, bool) {
	obs.cleanStaleHeapTop()
	return obs.priceHeap.Peek()
}

// GetBestLevel returns the price level at the best price
func (obs *OrderBookSide) GetBestLevel() *PriceLevel {
	price, ok := obs.GetBestPrice()
	if !ok {
		return nil
	}
	return obs.levels[price]
}

// IsEmpty returns true if there are no orders on this side
func (obs *OrderBookSide) IsEmpty() bool {
	return len(obs.levels) == 0
}

// StockOrderBook represents the order book for a single stock
type StockOrderBook struct {
	stock    string
	BuySide  *OrderBookSide // Bid side: buyers
	SellSide *OrderBookSide // Ask side: sellers
	Mu       sync.RWMutex   // Per-stock lock for concurrent access
}

// NewStockOrderBook creates a new order book for a stock
func NewStockOrderBook(stock string) *StockOrderBook {
	return &StockOrderBook{
		stock:    stock,
		BuySide:  NewOrderBookSide(true),
		SellSide: NewOrderBookSide(false),
	}
}
