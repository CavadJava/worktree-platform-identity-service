package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"platform-identity-service/internal/models"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already registered")
	ErrEmailTaken    = errors.New("email already registered")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	const q = `
		INSERT INTO users (id, name, username, email, password_hash, project_id, role_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, q,
		u.ID, u.Name, u.Username, u.Email, u.PasswordHash, u.ProjectID, u.RoleID, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_username_key" {
				return ErrUsernameTaken
			}
			return ErrEmailTaken
		}
		return err
	}
	return nil
}

const selectUserWithRole = `
	SELECT u.id, u.name, u.username, u.email, u.password_hash, u.project_id, u.role_id,
	       COALESCE(r.name, ''), u.created_at, u.updated_at
	FROM users u
	LEFT JOIN roles r ON r.id = u.role_id
`

func (r *UserRepository) scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash,
		&u.ProjectID, &u.RoleID, &u.RoleName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserWithRole+" WHERE u.id = $1", id)
	return r.scanUser(row)
}

func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserWithRole+" WHERE u.username = $1 OR u.email = $1", identifier)
	return r.scanUser(row)
}

func (r *UserRepository) CountByProject(ctx context.Context, projectID string) (int, error) {
	const q = `SELECT COUNT(*) FROM users WHERE project_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, q, projectID).Scan(&count)
	return count, err
}

func (r *UserRepository) SetRole(ctx context.Context, userID string, roleID int16) error {
	const q = `UPDATE users SET role_id = $2, updated_at = now() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, userID, roleID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}
