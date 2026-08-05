package repository

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrShopNotFound        = errors.New("shop not found")
	ErrProductItemNotFound = errors.New("product item not found")
)

// ProductItemInfo is what an order needs to know about the specific variant
// being purchased: which product it belongs to (and that product's shop,
// for the shop-match check), its display name, and the price actually
// charged (discount_price when the item is on sale, price otherwise).
type ProductItemInfo struct {
	ID          string
	ProductID   string
	ShopID      string
	ProductName string
	ItemName    string
	Price       float64
}

// LookupRepository does read-only cross-service lookups against tables
// owned by registration-service (`users`), shop-service (`shops`), and
// shop-product-service (`products`, `product_items`) — same shared
// Postgres instance, mirroring the established pattern of other services
// reading specific columns they don't own (e.g. shop-role-service reading
// `users`).
type LookupRepository struct {
	db *sql.DB
}

func NewLookupRepository(db *sql.DB) *LookupRepository {
	return &LookupRepository{db: db}
}

func (r *LookupRepository) GetUserSeq(ctx context.Context, userID string) (int64, error) {
	var seq int64
	err := r.db.QueryRowContext(ctx, `SELECT user_seq FROM users WHERE id = $1`, userID).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	return seq, err
}

func (r *LookupRepository) GetShopSeq(ctx context.Context, shopID string) (int64, error) {
	var seq int64
	err := r.db.QueryRowContext(ctx, `SELECT shop_seq FROM shops WHERE id = $1`, shopID).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrShopNotFound
	}
	return seq, err
}

// GetProductItem resolves the effective sale price server-side (never
// trusting a client-supplied price): discount_price when is_discounted is
// true, price otherwise.
func (r *LookupRepository) GetProductItem(ctx context.Context, productItemID string) (*ProductItemInfo, error) {
	const q = `
		SELECT pi.id, pi.product_id, p.shop_id, p.name, pi.name,
		       CASE WHEN pi.is_discounted AND pi.discount_price IS NOT NULL THEN pi.discount_price ELSE pi.price END
		FROM product_items pi
		JOIN products p ON p.id = pi.product_id
		WHERE pi.id = $1
	`
	info := &ProductItemInfo{}
	err := r.db.QueryRowContext(ctx, q, productItemID).Scan(
		&info.ID, &info.ProductID, &info.ShopID, &info.ProductName, &info.ItemName, &info.Price,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}
