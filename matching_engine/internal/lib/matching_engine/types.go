package matchingengine

import (
	"container/list"
	"math"
	"sort"
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
	orderId    string
	stock      string
	orderType  OrderType
	orderSide  OrderSide
	quantity   int
	limitPrice int
	timestamp  time.Time
}

// MatchedEvent represents a successful trade between buyer and seller
type MatchedEvent struct {
	buyerOrderId       string
	sellerOrderId      string
	pricePerStockCents int
	quantity           int
	timestamp          time.Time
}

// PriceLevel represents all orders at a specific price
type PriceLevel struct {
	price  int
	orders *list.List // Doubly-linked list for FIFO ordering
	volume int        // Total quantity at this price level
}

// NewPriceLevel creates a new price level
func NewPriceLevel(price int) *PriceLevel {
	return &PriceLevel{
		price:  price,
		orders: list.New(),
		volume: 0,
	}
}

// AddOrder adds an order to this price level (appends to end for FIFO)
func (pl *PriceLevel) AddOrder(order *Order) *list.Element {
	pl.volume += order.quantity
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
	pl.volume -= order.quantity
	pl.orders.Remove(element)
}

// RemoveFront removes and returns the first order
func (pl *PriceLevel) RemoveFront() *Order {
	if pl.orders.Len() == 0 {
		return nil
	}
	element := pl.orders.Front()
	order := element.Value.(*Order)
	pl.volume -= order.quantity
	pl.orders.Remove(element)
	return order
}

// IsEmpty checks if the price level has no orders
func (pl *PriceLevel) IsEmpty() bool {
	return pl.orders.Len() == 0
}

// OrderBookSide represents one side of the order book (buy or sell)
type OrderBookSide struct {
	levels       map[int]*PriceLevel      // price -> PriceLevel
	orderLookup  map[string]*list.Element // orderId -> list element for O(1) cancellation
	orderToPrice map[string]int           // orderId -> price for lookup
	isBuySide    bool                     // true for buy side, false for sell side
}

// NewOrderBookSide creates a new order book side
func NewOrderBookSide(isBuySide bool) *OrderBookSide {
	return &OrderBookSide{
		levels:       make(map[int]*PriceLevel),
		orderLookup:  make(map[string]*list.Element),
		orderToPrice: make(map[string]int),
		isBuySide:    isBuySide,
	}
}

// AddOrder adds an order to the order book side
func (obs *OrderBookSide) AddOrder(order *Order) {
	price := order.limitPrice

	// Create price level if it doesn't exist
	if _, exists := obs.levels[price]; !exists {
		obs.levels[price] = NewPriceLevel(price)
	}

	// Add order to price level
	element := obs.levels[price].AddOrder(order)
	obs.orderLookup[order.orderId] = element
	obs.orderToPrice[order.orderId] = price
}

// RemoveOrder removes an order by ID
func (obs *OrderBookSide) RemoveOrder(orderId string) bool {
	element, exists := obs.orderLookup[orderId]
	if !exists {
		return false
	}

	price := obs.orderToPrice[orderId]
	level := obs.levels[price]

	level.RemoveOrder(element)
	delete(obs.orderLookup, orderId)
	delete(obs.orderToPrice, orderId)

	// Remove empty price level
	if level.IsEmpty() {
		delete(obs.levels, price)
	}

	return true
}

// GetBestPrice returns the best price on this side (highest for buy, lowest for sell)
func (obs *OrderBookSide) GetBestPrice() (int, bool) {
	if len(obs.levels) == 0 {
		return 0, false
	}

	if obs.isBuySide {
		// Buy side: return highest price
		best := math.MinInt
		for price := range obs.levels {
			if price > best {
				best = price
			}
		}
		return best, true
	}

	// Sell side: return lowest price
	best := math.MaxInt
	for price := range obs.levels {
		if price < best {
			best = price
		}
	}
	return best, true
}

// GetBestLevel returns the price level at the best price
func (obs *OrderBookSide) GetBestLevel() *PriceLevel {
	price, ok := obs.GetBestPrice()
	if !ok {
		return nil
	}
	return obs.levels[price]
}

// GetSortedPrices returns all prices sorted (descending for buy, ascending for sell)
func (obs *OrderBookSide) GetSortedPrices() []int {
	prices := make([]int, 0, len(obs.levels))
	for price := range obs.levels {
		prices = append(prices, price)
	}

	if obs.isBuySide {
		sort.Sort(sort.Reverse(sort.IntSlice(prices))) // Highest first
	} else {
		sort.Ints(prices) // Lowest first
	}
	return prices
}

// IsEmpty returns true if there are no orders on this side
func (obs *OrderBookSide) IsEmpty() bool {
	return len(obs.levels) == 0
}

// StockOrderBook represents the order book for a single stock
type StockOrderBook struct {
	stock    string
	buySide  *OrderBookSide // Bid side: buyers
	sellSide *OrderBookSide // Ask side: sellers
	mu       sync.RWMutex   // Per-stock lock for concurrent access
}

// NewStockOrderBook creates a new order book for a stock
func NewStockOrderBook(stock string) *StockOrderBook {
	return &StockOrderBook{
		stock:    stock,
		buySide:  NewOrderBookSide(true),
		sellSide: NewOrderBookSide(false),
	}
}
