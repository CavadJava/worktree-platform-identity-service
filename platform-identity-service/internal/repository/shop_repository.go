package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrShopNotFound = errors.New("shop not found")

type ShopRepository struct {
	db *sql.DB
}

func NewShopRepository(db *sql.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

const selectShopColumns = `id, name, shop_type, contact_email, contact_phone, address, work_hours, created_at`

func scanShop(row *sql.Row) (*models.Shop, error) {
	var s models.Shop
	err := row.Scan(&s.ID, &s.Name, &s.ShopType, &s.ContactEmail, &s.ContactPhone, &s.Address, &s.WorkHours, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShopRepository) Create(ctx context.Context, s *models.Shop) error {
	const q = `INSERT INTO shops (id, name, shop_type, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.Name, s.ShopType, s.CreatedAt)
	return err
}

func (r *ShopRepository) List(ctx context.Context) ([]models.Shop, error) {
	q := `SELECT ` + selectShopColumns + ` FROM shops ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shops := []models.Shop{}
	for rows.Next() {
		var s models.Shop
		if err := rows.Scan(&s.ID, &s.Name, &s.ShopType, &s.ContactEmail, &s.ContactPhone, &s.Address, &s.WorkHours, &s.CreatedAt); err != nil {
			return nil, err
		}
		shops = append(shops, s)
	}
	return shops, rows.Err()
}

func (r *ShopRepository) GetByID(ctx context.Context, id string) (*models.Shop, error) {
	q := `SELECT ` + selectShopColumns + ` FROM shops WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanShop(row)
}

// ShopProfileUpdate carries the editable contact/profile fields — all four
// are always submitted together by the admin panel's profile form, so
// there's no partial-update variant here (unlike UserUpdate's nil-field
// pattern).
type ShopProfileUpdate struct {
	ContactEmail string
	ContactPhone string
	Address      string
	WorkHours    string
}

func (r *ShopRepository) UpdateProfile(ctx context.Context, id string, u ShopProfileUpdate) error {
	const q = `
		UPDATE shops SET contact_email = $2, contact_phone = $3, address = $4, work_hours = $5
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, q, id, u.ContactEmail, u.ContactPhone, u.Address, u.WorkHours)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrShopNotFound
	}
	return nil
}
