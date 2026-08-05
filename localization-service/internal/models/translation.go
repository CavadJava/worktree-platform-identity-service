package models

import "time"

type Translation struct {
	ID        string    `json:"id"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Locale    string    `json:"locale"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
