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

const (
	ShopTypeForeign = "foreign"
	ShopTypeLocal   = "local"
)

type Shop struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ShopType     string    `json:"shop_type"`
	ContactEmail string    `json:"contact_email"`
	ContactPhone string    `json:"contact_phone"`
	Address      string    `json:"address"`
	WorkHours    string    `json:"work_hours"`
	CreatedAt    time.Time `json:"created_at"`
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
	Phone          *string // optional — nil for any account that didn't provide one
	PasswordHash   string
	PlainPassword  *string // see postgres.go's plain_password column comment
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
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	TechStack     string    `json:"tech_stack"`
	AutoSubscribe bool      `json:"auto_subscribe"`
	CreatedAt     time.Time `json:"created_at"`
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

// SubscriptionWithUser is a Subscription joined with its user's identity
// fields, for a product's "customers" list — an admin/superadmin viewing
// who has access to a product needs the person's name/username/email/role
// alongside the subscription itself, not just the bare user_id.
type SubscriptionWithUser struct {
	Subscription
	UserName       string
	UserUsername   string
	UserEmail      string
	UserSystemRole string
}
