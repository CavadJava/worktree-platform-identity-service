package models

import "time"

const (
	SystemRoleSuperadmin = "superadmin"
	SystemRoleAdmin      = "admin"
	SystemRoleUser       = "user"
)

const (
	ShopRoleAdmin = "shop-admin"
	ShopRoleUser  = "shop-user"
)

const (
	UserStatusActive   = "ACTIVE"
	UserStatusInActive = "IN_ACTIVE"
)

type Shop struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemRole struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

type ShopRole struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID             string
	Name           string
	Username       string
	Email          string
	PasswordHash   string
	SystemRoleID   int16
	SystemRoleName string // populated by joined queries
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ShopMembership is a row from user_shop_memberships, joined with the
// shop's name and the shop role's name for direct display — mirrors the
// old User.RoleName join-only-field precedent.
type ShopMembership struct {
	ID           string
	UserID       string
	ShopID       string
	ShopName     string
	ShopRoleID   int16
	ShopRoleName string
	CreatedAt    time.Time
}

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TechStack   string    `json:"tech_stack"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProductSubproject struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	ProductAdminRequestPending  = "pending"
	ProductAdminRequestApproved = "approved"
	ProductAdminRequestRejected = "rejected"
)

type ProductAdminRequest struct {
	ID                string
	ProductID         string
	SubjectUserID     string
	RequestedByUserID string
	Status            string
	CreatedAt         time.Time
	DecidedAt         *time.Time
	DecidedByUserID   *string
}

type Subscription struct {
	ID          string
	UserID      string
	ProductID   string
	Subscripted bool
	Renewed     bool
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
