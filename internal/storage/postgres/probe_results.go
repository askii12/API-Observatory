package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/askii12/api-observatory/internal/domain"
)

type ProbeResultRepository struct {
	db *sql.DB
}

func NewProbeResultRepository(db *sql.DB) *ProbeResultRepository {
	return &ProbeResultRepository{db: db}
}

func (r *ProbeResultRepository) Create(
	ctx context.Context,
	result domain.ProbeResult,
) error {
	const query = `
		INSERT INTO probe_results (
			monitor_id,
			status,
			status_code,
			latency_ms,
			error_message,
			checked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		result.MonitorID,
		result.Status,
		result.StatusCode,
		result.LatencyMs,
		result.ErrorMessage,
		result.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("create probe result: %w", err)
	}

	return nil
}

func (r *ProbeResultRepository) ListByMonitorID(
	ctx context.Context,
	monitorID string,
	limit int,
) ([]domain.ProbeResult, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	const query = `
		SELECT
			id,
			monitor_id,
			status,
			status_code,
			latency_ms,
			error_message,
			checked_at
		FROM probe_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC, id DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list probe results: %w", err)
	}
	defer rows.Close()

	results := make([]domain.ProbeResult, 0)

	for rows.Next() {
		var result domain.ProbeResult

		if err := rows.Scan(
			&result.ID,
			&result.MonitorID,
			&result.Status,
			&result.StatusCode,
			&result.LatencyMs,
			&result.ErrorMessage,
			&result.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("scan probe result: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate probe results: %w", err)
	}

	return results, nil
}

var _ domain.ProbeResultRepository = (*ProbeResultRepository)(nil)
