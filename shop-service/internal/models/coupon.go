package models

import "time"

type Coupon struct {
	ID              string     `json:"id"`
	Code            string     `json:"code"`
	Title           string     `json:"title"`
	DiscountPercent int        `json:"discount_percent"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UserCoupon struct {
	Coupon
	ClaimedAt time.Time `json:"claimed_at"`
}
