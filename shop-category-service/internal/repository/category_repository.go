package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"shop-category-service/internal/models"
)

var (
	ErrCategoryNotFound    = errors.New("category not found")
	ErrSubcategoryNotFound = errors.New("subcategory not found")
	ErrIDExists            = errors.New("id already exists")
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func isDuplicate(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}

// ---- Categories ----

func (r *CategoryRepository) CreateCategory(ctx context.Context, c *models.Category) error {
	const q = `
		INSERT INTO categories (id, name, icon, sort_order)
		VALUES ($1, $2, NULLIF($3, ''), $4)
	`
	_, err := r.db.ExecContext(ctx, q, c.ID, c.Name, c.Icon, c.SortOrder)
	if isDuplicate(err) {
		return ErrIDExists
	}
	return err
}

func (r *CategoryRepository) FindCategory(ctx context.Context, id string) (*models.Category, error) {
	const q = `
		SELECT id, name, COALESCE(icon, ''), sort_order, created_at, updated_at
		FROM categories WHERE id = $1
	`
	c := &models.Category{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CategoryRepository) ListCategories(ctx context.Context) ([]*models.Category, error) {
	const q = `
		SELECT id, name, COALESCE(icon, ''), sort_order, created_at, updated_at
		FROM categories ORDER BY sort_order ASC, name ASC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []*models.Category{}
	for rows.Next() {
		c := &models.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, id, name, icon string, sortOrder int) (*models.Category, error) {
	const q = `
		UPDATE categories SET name = $2, icon = NULLIF($3, ''), sort_order = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, name, COALESCE(icon, ''), sort_order, created_at, updated_at
	`
	c := &models.Category{}
	err := r.db.QueryRowContext(ctx, q, id, name, icon, sortOrder).Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CategoryRepository) DeleteCategory(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// ---- Subcategories ----

func (r *CategoryRepository) CreateSubcategory(ctx context.Context, s *models.Subcategory) error {
	const q = `
		INSERT INTO subcategories (id, category_id, name, image, sort_order)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
	`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.CategoryID, s.Name, s.Image, s.SortOrder)
	if isDuplicate(err) {
		return ErrIDExists
	}
	return err
}

func (r *CategoryRepository) FindSubcategory(ctx context.Context, id string) (*models.Subcategory, error) {
	const q = `
		SELECT id, category_id, name, COALESCE(image, ''), sort_order, created_at, updated_at
		FROM subcategories WHERE id = $1
	`
	s := &models.Subcategory{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&s.ID, &s.CategoryID, &s.Name, &s.Image, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubcategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CategoryRepository) ListSubcategories(ctx context.Context, categoryID string) ([]*models.Subcategory, error) {
	q := `
		SELECT id, category_id, name, COALESCE(image, ''), sort_order, created_at, updated_at
		FROM subcategories
	`
	args := []interface{}{}
	if categoryID != "" {
		q += ` WHERE category_id = $1`
		args = append(args, categoryID)
	}
	q += ` ORDER BY sort_order ASC, name ASC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subcategories := []*models.Subcategory{}
	for rows.Next() {
		s := &models.Subcategory{}
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.Image, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subcategories = append(subcategories, s)
	}
	return subcategories, rows.Err()
}

func (r *CategoryRepository) UpdateSubcategory(ctx context.Context, id, name, image string, sortOrder int) (*models.Subcategory, error) {
	const q = `
		UPDATE subcategories SET name = $2, image = NULLIF($3, ''), sort_order = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, category_id, name, COALESCE(image, ''), sort_order, created_at, updated_at
	`
	s := &models.Subcategory{}
	err := r.db.QueryRowContext(ctx, q, id, name, image, sortOrder).Scan(&s.ID, &s.CategoryID, &s.Name, &s.Image, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubcategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *CategoryRepository) DeleteSubcategory(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM subcategories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSubcategoryNotFound
	}
	return nil
}
