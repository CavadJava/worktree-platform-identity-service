package repository

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrShopNotFound    = errors.New("shop not found")
	ErrProductNotFound = errors.New("product not found")
)

type ProductInfo struct {
	ID     string
	ShopID string
	Name   string
	Price  float64
}

// LookupRepository does read-only cross-service lookups against tables
// owned by registration-service (`users`), shop-service (`shops`), and
// shop-product-service (`products`) — same shared Postgres instance,
// mirroring the established pattern of other services reading specific
// columns they don't own (e.g. shop-role-service reading `users`).
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

func (r *LookupRepository) GetProduct(ctx context.Context, productID string) (*ProductInfo, error) {
	const q = `SELECT id, shop_id, name, price FROM products WHERE id = $1`
	p := &ProductInfo{}
	err := r.db.QueryRowContext(ctx, q, productID).Scan(&p.ID, &p.ShopID, &p.Name, &p.Price)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
