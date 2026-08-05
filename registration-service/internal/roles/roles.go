// Package roles defines the role taxonomy shared (by convention, not by
// import — each service keeps its own copy) across registration-service,
// authorization-service, shop-role-service, shop-service and
// shop-product-service.
//
//   - Role is the account's base role: every user gets "user"; "administrator"
//     manages both regular users and the shopping domain (assigns shop levels,
//     approves shop applications).
//   - ShopRoleLevel is a single hierarchical level an account holds within the
//     one shop it belongs to (ShopID). Higher levels include every permission
//     of the levels below them: admin(4) > review(3) > add-product(2) > chat(1).
package roles

const (
	RoleUser          = "user"
	RoleAdministrator = "administrator"
)

const (
	ShopLevelNone       = 0
	ShopLevelChat       = 1
	ShopLevelAddProduct = 2
	ShopLevelReview     = 3
	ShopLevelAdmin      = 4
)

func IsValidShopLevel(level int) bool {
	return level >= ShopLevelChat && level <= ShopLevelAdmin
}
