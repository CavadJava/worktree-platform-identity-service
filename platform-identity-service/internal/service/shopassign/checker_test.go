package shopassign

import (
	"testing"

	"platform-identity-service/internal/models"
)

func TestCanManageShop(t *testing.T) {
	tests := []struct {
		name       string
		caller     Caller
		membership *models.ShopMembership
		want       bool
	}{
		{
			name:       "superadmin can manage any shop, even with no membership",
			caller:     Caller{UserID: "u1", SystemRole: "superadmin"},
			membership: nil,
			want:       true,
		},
		{
			name:       "shop-admin member can manage their own shop",
			caller:     Caller{UserID: "u1", SystemRole: "user"},
			membership: &models.ShopMembership{UserID: "u1", ShopID: "s1", ShopRoleName: "shop-admin"},
			want:       true,
		},
		{
			name:       "shop-user member cannot manage the shop",
			caller:     Caller{UserID: "u1", SystemRole: "user"},
			membership: &models.ShopMembership{UserID: "u1", ShopID: "s1", ShopRoleName: "shop-user"},
			want:       false,
		},
		{
			name:       "non-member, non-superadmin cannot manage the shop",
			caller:     Caller{UserID: "u1", SystemRole: "user"},
			membership: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanManageShop(tt.caller, tt.membership)
			if got != tt.want {
				t.Errorf("CanManageShop(%+v, %+v) = %v, want %v", tt.caller, tt.membership, got, tt.want)
			}
		})
	}
}
