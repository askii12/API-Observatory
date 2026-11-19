package domain

import (
	"context"
	"time"
)

type Monitor struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	TimeoutMs       int       `json:"timeout_ms"`
	Enabled         bool      `json:"enabled"`
	NextRunAt       time.Time `json:"next_run_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MonitorRepository interface {
	Create(ctx context.Context, monitor *Monitor) error
	List(ctx context.Context) ([]Monitor, error)
	GetByID(ctx context.Context, id string) (*Monitor, error)
	Update(ctx context.Context, monitor *Monitor) error
	Delete(ctx context.Context, id string) error
}
