package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"payment-service/internal/models"
)

var (
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentAlreadyExists = errors.New("a payment already exists for this order")
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts the payment + its line items and credits the shop's
// balance, all in one transaction — a shop's balance must never drift from
// the sum of its completed payments.
func (r *PaymentRepository) Create(ctx context.Context, p *models.Payment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	const insertPayment = `
		INSERT INTO payments (id, order_id, user_id, shop_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if _, err := tx.ExecContext(ctx, insertPayment, p.ID, p.OrderID, p.UserID, p.ShopID, p.Amount, p.Status, p.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrPaymentAlreadyExists
		}
		return err
	}

	const insertItem = `
		INSERT INTO payment_items (id, payment_id, product_id, product_name, quantity, unit_price)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, item := range p.Items {
		if _, err := tx.ExecContext(ctx, insertItem, item.ID, p.ID, item.ProductID, item.ProductName, item.Quantity, item.UnitPrice); err != nil {
			return err
		}
	}

	const upsertBalance = `
		INSERT INTO shop_balances (shop_id, balance, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (shop_id) DO UPDATE SET balance = shop_balances.balance + EXCLUDED.balance, updated_at = now()
	`
	if _, err := tx.ExecContext(ctx, upsertBalance, p.ShopID, p.Amount); err != nil {
		return err
	}

	return tx.Commit()
}

// ListByShop returns a shop's payments newest-first, each with its line
// items attached — two queries (payments, then their items in one IN
// query) rather than a join, mirroring review-service's ListByProduct.
func (r *PaymentRepository) ListByShop(ctx context.Context, shopID string) ([]*models.Payment, error) {
	const q = `
		SELECT id, order_id, user_id, shop_id, amount, status, created_at
		FROM payments WHERE shop_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := []*models.Payment{}
	ids := make([]string, 0)
	byID := map[string]*models.Payment{}
	for rows.Next() {
		p := &models.Payment{Items: []models.PaymentItem{}}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.UserID, &p.ShopID, &p.Amount, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
		ids = append(ids, p.ID)
		byID[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return payments, nil
	}

	itemRows, err := r.db.QueryContext(ctx, `
		SELECT id, payment_id, product_id, product_name, quantity, unit_price
		FROM payment_items WHERE payment_id::text = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		item := models.PaymentItem{}
		if err := itemRows.Scan(&item.ID, &item.PaymentID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, err
		}
		if p, ok := byID[item.PaymentID]; ok {
			p.Items = append(p.Items, item)
		}
	}
	return payments, itemRows.Err()
}

// GetBalance returns a shop's balance, or a zero balance if it has never
// received a payment — callers shouldn't have to special-case "no row yet".
func (r *PaymentRepository) GetBalance(ctx context.Context, shopID string) (*models.ShopBalance, error) {
	const q = `SELECT shop_id, balance, updated_at FROM shop_balances WHERE shop_id = $1`
	b := &models.ShopBalance{}
	err := r.db.QueryRowContext(ctx, q, shopID).Scan(&b.ShopID, &b.Balance, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &models.ShopBalance{ShopID: shopID, Balance: 0}, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}
