package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *MonitorRepository) ClaimDue(
	ctx context.Context,
	workerID string,
	limit int,
	leaseDuration time.Duration,
) ([]domain.Monitor, error) {
	workerID = strings.TrimSpace(workerID)

	if workerID == "" {
		return nil, fmt.Errorf("worker ID must not be empty")
	}

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	if leaseDuration < time.Microsecond {
		return nil, fmt.Errorf("lease duration must be at least one microsecond")
	}

	const query = `
        WITH due_monitors AS (
            SELECT id
            FROM monitors
            WHERE enabled = TRUE
              AND next_run_at <= NOW()
              AND (
                  lease_expires_at IS NULL
                  OR lease_expires_at <= NOW()
              )
            ORDER BY next_run_at ASC, id ASC
            LIMIT $1
            FOR UPDATE SKIP LOCKED
        )
        UPDATE monitors AS m
        SET
            lease_owner = $2,
            lease_expires_at = NOW() + ($3 * INTERVAL '1 microsecond')
        FROM due_monitors AS due
        WHERE m.id = due.id
        RETURNING
            m.id,
            m.name,
            m.url,
            m.interval_seconds,
            m.timeout_ms,
            m.enabled,
            m.next_run_at,
            m.created_at,
            m.updated_at
    `

	rows, err := r.db.QueryContext(
		ctx,
		query,
		limit,
		workerID,
		leaseDuration.Microseconds(),
	)
	if err != nil {
		return nil, fmt.Errorf("claim due monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]domain.Monitor, 0)

	for rows.Next() {
		var monitor domain.Monitor

		if err := rows.Scan(
			&monitor.ID,
			&monitor.Name,
			&monitor.URL,
			&monitor.IntervalSeconds,
			&monitor.TimeoutMs,
			&monitor.Enabled,
			&monitor.NextRunAt,
			&monitor.CreatedAt,
			&monitor.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed monitor: %w", err)
		}

		monitors = append(monitors, monitor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed monitors: %w", err)
	}

	return monitors, nil
}
