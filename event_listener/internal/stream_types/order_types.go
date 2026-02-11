package streamtypes

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
