package models

import "time"

type Product struct {
	ID             string    `json:"id"`
	ShopID         string    `json:"shop_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Price          float64   `json:"price"`
	Stock          int       `json:"stock"`
	ProductTypeID  *string   `json:"product_type_id,omitempty"`
	Brand          *string   `json:"brand,omitempty"`
	HasBrand       bool      `json:"has_brand"`
	Material       *string   `json:"material,omitempty"`
	HasMaterial    bool      `json:"has_material"`
	WeightKg       *float64  `json:"weight_kg,omitempty"`
	HasWeight      bool      `json:"has_weight"`
	OriginCountry  *string   `json:"origin_country,omitempty"`
	HasOrigin      bool      `json:"has_origin"`
	WarrantyMonths *int      `json:"warranty_months,omitempty"`
	HasWarranty    bool      `json:"has_warranty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ComputeFlags derives the Has* presence flags from their underlying
// pointer fields — never persisted, always recomputed after a Scan/Create/
// Update so they can't drift from the values they describe. A blank string
// counts as "not set", same as nil.
func (p *Product) ComputeFlags() {
	p.HasBrand = p.Brand != nil && *p.Brand != ""
	p.HasMaterial = p.Material != nil && *p.Material != ""
	p.HasWeight = p.WeightKg != nil
	p.HasOrigin = p.OriginCountry != nil && *p.OriginCountry != ""
	p.HasWarranty = p.WarrantyMonths != nil
}

// ProductDetails groups the optional "deep" attributes — supplied together
// at both Create and Update call sites, kept separate from Product's core
// fields (name/price/stock) so those signatures don't grow a 6th+ param.
type ProductDetails struct {
	Brand          *string
	Material       *string
	WeightKg       *float64
	OriginCountry  *string
	WarrantyMonths *int
}
