package models

import "time"

// StatusCompleted is the only status this mock payment flow ever produces —
// there's no real payment gateway to fail or stay pending on. Kept as a
// string column (not a hardcoded bool) so a future refund/payout-related
// status can be added without a schema change.
const StatusCompleted = "completed"

// Payment is one order's settlement — 1:1 with shop-order-service's Order
// (an order is always single-shop, so a payment is too). Crediting a
// shop's ShopBalance happens atomically alongside creating this row (see
// PaymentRepository.Create).
type Payment struct {
	ID        string        `json:"id"`
	OrderID   string        `json:"order_id"`
	UserID    string        `json:"user_id"`
	ShopID    string        `json:"shop_id"`
	Amount    float64       `json:"amount"`
	Status    string        `json:"status"`
	Items     []PaymentItem `json:"items"`
	CreatedAt time.Time     `json:"created_at"`
}

// PaymentItem snapshots a purchased line item at payment time (mirrors
// shop-order-service's OrderItem snapshot) so a shop's payment history is
// self-contained — no cross-service call needed to show what was sold.
type PaymentItem struct {
	ID          string  `json:"id"`
	PaymentID   string  `json:"payment_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// ShopBalance is a shop's running "temporary" ledger — every completed
// payment adds Amount to it. It exists purely inside payment-service today;
// transferring it out to a shop's real bank account is a future feature
// this schema deliberately doesn't block (a payout would just subtract from
// Balance and record its own ledger entry — not built yet, see README).
type ShopBalance struct {
	ShopID    string    `json:"shop_id"`
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}
