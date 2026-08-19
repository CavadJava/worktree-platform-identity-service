// Package shopassign isolates the authorization decision for who may
// manage a shop's membership (add members, change a member's shop role).
package shopassign

import "platform-identity-service/internal/models"

type Caller struct {
	UserID     string
	SystemRole string
}

// CanManageShop reports whether caller may add/modify members of the shop
// membership belongs to. membership is the caller's own
// user_shop_memberships row for that shop, or nil if they have none.
// A superadmin always may, regardless of membership. Otherwise the caller
// must hold a shop-admin membership row for that specific shop.
func CanManageShop(caller Caller, membership *models.ShopMembership) bool {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return true
	}
	if membership == nil {
		return false
	}
	return membership.ShopRoleName == models.ShopRoleAdmin
}
