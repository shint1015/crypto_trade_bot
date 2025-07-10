package domain

import "time"

// OrderSide は注文のサイド（買い/売り）を表します。
type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

// OrderStatus は注文の状態を表します。
type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
)

// Order は取引注文の情報を保持するエンティティです。
type Order struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Price     float64
	Amount    float64
	Status    OrderStatus
	CreatedAt time.Time
}
