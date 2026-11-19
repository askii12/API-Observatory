package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/askii12/api-observatory/internal/domain"
)

type MonitorRepository struct {
	db *sql.DB
}

func NewMonitorRepository(db *sql.DB) *MonitorRepository {
	return &MonitorRepository{db: db}
}

func (r *MonitorRepository) Create(
	ctx context.Context,
	monitor *domain.Monitor,
) error {
	query := `
		INSERT INTO monitors (
			name, url, interval_seconds, timeout_ms, enabled
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id, next_run_at, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		monitor.Name,
		monitor.URL,
		monitor.IntervalSeconds,
		monitor.TimeoutMs,
		monitor.Enabled,
	).Scan(
		&monitor.ID,
		&monitor.NextRunAt,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)
}

func (r *MonitorRepository) List(
	ctx context.Context,
) ([]domain.Monitor, error) {
	query := `
		SELECT
			id, name, url, interval_seconds,
			timeout_ms, enabled, next_run_at,
			created_at, updated_at
		FROM monitors
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := make([]domain.Monitor, 0)

	for rows.Next() {
		var monitor domain.Monitor

		err := rows.Scan(
			&monitor.ID,
			&monitor.Name,
			&monitor.URL,
			&monitor.IntervalSeconds,
			&monitor.TimeoutMs,
			&monitor.Enabled,
			&monitor.NextRunAt,
			&monitor.CreatedAt,
			&monitor.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		monitors = append(monitors, monitor)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return monitors, nil
}

func (r *MonitorRepository) GetByID(
	ctx context.Context,
	id string,
) (*domain.Monitor, error) {
	query := `
		SELECT
			id, name, url, interval_seconds,
			timeout_ms, enabled, next_run_at,
			created_at, updated_at
		FROM monitors
		WHERE id::text = $1
	`

	var monitor domain.Monitor

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&monitor.ID,
		&monitor.Name,
		&monitor.URL,
		&monitor.IntervalSeconds,
		&monitor.TimeoutMs,
		&monitor.Enabled,
		&monitor.NextRunAt,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	return &monitor, nil
}

func (r *MonitorRepository) Update(
	ctx context.Context,
	monitor *domain.Monitor,
) error {
	query := `
		UPDATE monitors
		SET
			name = $1,
			url = $2,
			interval_seconds = $3,
			timeout_ms = $4,
			enabled = $5,
			updated_at = NOW()
		WHERE id::text = $6
		RETURNING
			id, name, url, interval_seconds,
			timeout_ms, enabled, next_run_at,
			created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		monitor.Name,
		monitor.URL,
		monitor.IntervalSeconds,
		monitor.TimeoutMs,
		monitor.Enabled,
		monitor.ID,
	).Scan(
		&monitor.ID,
		&monitor.Name,
		&monitor.URL,
		&monitor.IntervalSeconds,
		&monitor.TimeoutMs,
		&monitor.Enabled,
		&monitor.NextRunAt,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)
}

func (r *MonitorRepository) Delete(
	ctx context.Context,
	id string,
) error {
	query := `DELETE FROM monitors WHERE id::text = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
