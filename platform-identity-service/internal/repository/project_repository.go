package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrProjectNotFound = errors.New("project not found")

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, p *models.Project) error {
	const q = `INSERT INTO projects (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Name, p.CreatedAt)
	return err
}

func (r *ProjectRepository) List(ctx context.Context) ([]models.Project, error) {
	const q = `SELECT id, name, created_at FROM projects ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []models.Project{}
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*models.Project, error) {
	const q = `SELECT id, name, created_at FROM projects WHERE id = $1`
	var p models.Project
	err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Name, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
