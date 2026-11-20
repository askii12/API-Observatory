package domain

import "context"

type ProbeResultRepository interface {
	Create(ctx context.Context, result ProbeResult) error

	ListByMonitorID(
		ctx context.Context,
		monitorID string,
		limit int,
	) ([]ProbeResult, error)
}
