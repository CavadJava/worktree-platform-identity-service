package models

import "time"

// Role name constants — single source of truth for the two role names
// used across handlers, services, and authorization strategies.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID           string
	Name         string
	Username     string
	Email        string
	PasswordHash string
	ProjectID    *string
	RoleID       *int16
	RoleName     string // populated by joined queries, not persisted directly
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
