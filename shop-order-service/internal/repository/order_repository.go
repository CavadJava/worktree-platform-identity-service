package repository

import (
	"context"
	"database/sql"
	"errors"

	"shop-order-service/internal/models"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts the order and its items in one transaction, so a failure
// partway through never leaves an order with no items (or vice versa).
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const orderQ = `
		INSERT INTO orders (id, order_number, user_id, shop_id, status, total_amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.ExecContext(ctx, orderQ,
		order.ID, order.OrderNumber, order.UserID, order.ShopID, order.Status, order.TotalAmount, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return err
	}

	const itemQ = `
		INSERT INTO order_items (id, order_id, product_id, product_name, unit_price, quantity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for _, item := range order.Items {
		if _, err := tx.ExecContext(ctx, itemQ, item.ID, order.ID, item.ProductID, item.ProductName, item.UnitPrice, item.Quantity, item.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CountByUser is used to compute the next order_number segment for a user
// (their Nth order overall) — call it inside the same request as Create,
// before building the order, to minimize (not eliminate) the race window.
func (r *OrderRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*models.Order, error) {
	const q = `
		SELECT id, order_number, user_id, shop_id, status, total_amount, created_at, updated_at
		FROM orders WHERE id = $1
	`
	o := &models.Order{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&o.ID, &o.OrderNumber, &o.UserID, &o.ShopID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	items, err := r.itemsFor(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID string) ([]*models.Order, error) {
	const q = `
		SELECT id, order_number, user_id, shop_id, status, total_amount, created_at, updated_at
		FROM orders WHERE user_id = $1
		ORDER BY created_at DESC
	`
	return r.list(ctx, q, userID)
}

func (r *OrderRepository) ListByShop(ctx context.Context, shopID string) ([]*models.Order, error) {
	const q = `
		SELECT id, order_number, user_id, shop_id, status, total_amount, created_at, updated_at
		FROM orders WHERE shop_id = $1
		ORDER BY created_at DESC
	`
	return r.list(ctx, q, shopID)
}

func (r *OrderRepository) list(ctx context.Context, q, arg string) ([]*models.Order, error) {
	rows, err := r.db.QueryContext(ctx, q, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*models.Order{}
	for rows.Next() {
		o := &models.Order{}
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.ShopID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) itemsFor(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	const q = `
		SELECT id, order_id, product_id, product_name, unit_price, quantity, created_at
		FROM order_items WHERE order_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.OrderItem{}
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.UnitPrice, &item.Quantity, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
