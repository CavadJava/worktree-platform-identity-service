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
		INSERT INTO users (id, name, username, email, password_hash, system_role_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, q,
		u.ID, u.Name, u.Username, u.Email, u.PasswordHash, u.SystemRoleID, u.Status, u.CreatedAt, u.UpdatedAt,
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

const selectUserWithSystemRole = `
	SELECT u.id, u.name, u.username, u.email, u.password_hash, u.system_role_id, sr.name, u.status, u.created_at, u.updated_at
	FROM users u
	JOIN system_roles sr ON sr.id = u.system_role_id
`

func (r *UserRepository) scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash,
		&u.SystemRoleID, &u.SystemRoleName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserWithSystemRole+" WHERE u.id = $1", id)
	return r.scanUser(row)
}

func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, selectUserWithSystemRole+" WHERE u.username = $1 OR u.email = $1", identifier)
	return r.scanUser(row)
}

func (r *UserRepository) ListAll(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, selectUserWithSystemRole+" ORDER BY u.created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash,
			&u.SystemRoleID, &u.SystemRoleName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) SetSystemRole(ctx context.Context, userID string, systemRoleID int16) error {
	const q = `UPDATE users SET system_role_id = $2, updated_at = now() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, userID, systemRoleID)
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

// UserUpdate carries the optional fields Update may change — a nil field
// leaves that column untouched, so callers only pass what they're
// actually changing (name/email/password can each be edited independently).
type UserUpdate struct {
	Name         *string
	Email        *string
	PasswordHash *string
}

func (r *UserRepository) Update(ctx context.Context, userID string, u UserUpdate) error {
	const q = `
		UPDATE users SET
			name = COALESCE($2, name),
			email = COALESCE($3, email),
			password_hash = COALESCE($4, password_hash),
			updated_at = now()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, q, userID, u.Name, u.Email, u.PasswordHash)
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

func (r *UserRepository) SetStatus(ctx context.Context, userID, status string) error {
	const q = `UPDATE users SET status = $2, updated_at = now() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, userID, status)
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
