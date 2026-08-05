package repository

import (
	"context"
	"database/sql"
	"errors"

	"notification-service/internal/models"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	const q = `
		INSERT INTO notifications (id, user_id, type, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q, n.ID, n.UserID, n.Type, n.Message, n.IsRead, n.CreatedAt)
	return err
}

func (r *NotificationRepository) ListByUser(ctx context.Context, userID string, limit int) ([]*models.Notification, error) {
	const q = `
		SELECT id, user_id, type, message, is_read, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, q, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []*models.Notification{}
	for rows.Next() {
		n := &models.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID,
	).Scan(&count)
	return count, err
}

// MarkRead flips is_read for the user's own notification — the user_id
// guard keeps one user from marking another's notifications.
func (r *NotificationRepository) MarkRead(ctx context.Context, userID, id string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID,
	)
	return err
}
