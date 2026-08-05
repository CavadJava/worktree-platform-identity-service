package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"log-service/internal/models"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Create(ctx context.Context, e *models.LogEntry) error {
	const q = `
		INSERT INTO service_logs (id, service, level, method, path, status, duration_ms, message, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, NULLIF($8, ''), $9)
	`
	_, err := r.db.ExecContext(ctx, q, e.ID, e.Service, e.Level, e.Method, e.Path, e.Status, e.DurationMs, e.Message, e.CreatedAt)
	return err
}

type LogFilter struct {
	Service string
	Level   string
	Path    string
	From    *time.Time
	To      *time.Time
	Limit   int
}

func (r *LogRepository) List(ctx context.Context, f LogFilter) ([]*models.LogEntry, error) {
	q := `
		SELECT id, service, level, COALESCE(method, ''), COALESCE(path, ''),
		       COALESCE(status, 0), COALESCE(duration_ms, 0), COALESCE(message, ''), created_at
		FROM service_logs
		WHERE 1=1
	`
	args := []interface{}{}
	i := 1
	if f.Service != "" {
		q += " AND service = $" + strconv.Itoa(i)
		args = append(args, f.Service)
		i++
	}
	if f.Level != "" {
		q += " AND level = $" + strconv.Itoa(i)
		args = append(args, f.Level)
		i++
	}
	if f.Path != "" {
		q += " AND path ILIKE $" + strconv.Itoa(i)
		args = append(args, "%"+f.Path+"%")
		i++
	}
	if f.From != nil {
		q += " AND created_at >= $" + strconv.Itoa(i)
		args = append(args, *f.From)
		i++
	}
	if f.To != nil {
		q += " AND created_at <= $" + strconv.Itoa(i)
		args = append(args, *f.To)
		i++
	}
	q += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(i)
	args = append(args, f.Limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*models.LogEntry{}
	for rows.Next() {
		e := &models.LogEntry{}
		if err := rows.Scan(&e.ID, &e.Service, &e.Level, &e.Method, &e.Path, &e.Status, &e.DurationMs, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *LogRepository) ListServices(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT service FROM service_logs ORDER BY service`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	services := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}
