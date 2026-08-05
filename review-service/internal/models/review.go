package models

import "time"

// Review is a shopper's rating/text for a shop-product-service product,
// with zero or more attached photos/clips.
type Review struct {
	ID        string        `json:"id"`
	ProductID string        `json:"product_id"`
	UserID    string        `json:"user_id"`
	Rating    int           `json:"rating"`
	Text      string        `json:"text,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Media     []ReviewMedia `json:"media"`
}

// ReviewMedia is one uploaded photo or clip attached to a Review. URL is a
// path relative to this service (e.g. "/media/reviews/<id>/<file>") — the
// client prepends its own host, same as it does for every API call.
type ReviewMedia struct {
	ID        string    `json:"id"`
	ReviewID  string    `json:"review_id"`
	MediaType string    `json:"media_type"` // "image" | "video"
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}
