// Package roles mirrors the role taxonomy owned by registration-service
// (see its internal/roles package for the full doc).
package roles

const RoleAdministrator = "administrator"

const (
	ShopLevelChat       = 1
	ShopLevelAddProduct = 2
	ShopLevelReview     = 3
	ShopLevelAdmin      = 4
)
