package models

import "time"

const (
	ApplicationStatusPending  = "pending"   // initial request, no shop yet
	ApplicationStatusFormSent = "form_sent" // administrator sent the form: a temporary shop exists, applicant is its admin(4) and fills in the details
	ApplicationStatusApproved = "approved"
	ApplicationStatusRejected = "rejected"
)

type ShopApplication struct {
	ID              string    `json:"id"`
	ApplicantID     string    `json:"applicant_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	TemplateVersion string    `json:"template_version"`
	Status          string    `json:"status"`
	ReviewerID      *string   `json:"reviewer_id,omitempty"`
	ReviewNote      string    `json:"review_note,omitempty"`
	ShopID          *string   `json:"shop_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
