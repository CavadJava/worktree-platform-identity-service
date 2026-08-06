package models

import "time"

// Subscriber is one row of a shop's subscriber list — deliberately just the
// user id + when they subscribed. Resolving a display name would need
// registration-service's admin-only GET /users/{id}, so shop staff (who
// aren't administrators) couldn't use it anyway.
type Subscriber struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
