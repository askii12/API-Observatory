package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
)

func createTestMonitor(
	t *testing.T,
	repo *MonitorRepository,
) *domain.Monitor {
	t.Helper()

	monitor := &domain.Monitor{
		Name:            "Probe Result Integration Test",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutMs:       5000,
		Enabled:         true,
	}

	if err := repo.Create(context.Background(), monitor); err != nil {
		t.Fatalf("create test monitor: %v", err)
	}

	return monitor
}

func TestProbeResultRepositoryCreate(t *testing.T) {
	db, monitorRepo := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	monitor := createTestMonitor(t, monitorRepo)

	statusCode := 200
	result := domain.ProbeResult{
		MonitorID:  monitor.ID,
		Status:     domain.ProbeStatusSuccess,
		StatusCode: &statusCode,
		LatencyMs:  125,
		CheckedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}

	if err := repo.Create(context.Background(), result); err != nil {
		t.Fatalf("create probe result: %v", err)
	}

	results, err := repo.ListByMonitorID(
		context.Background(),
		monitor.ID,
		10,
	)
	if err != nil {
		t.Fatalf("list probe results: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]

	if got.ID == "" {
		t.Fatal("expected generated probe result ID")
	}

	if got.MonitorID != monitor.ID {
		t.Errorf("expected monitor ID %q, got %q", monitor.ID, got.MonitorID)
	}

	if got.Status != domain.ProbeStatusSuccess {
		t.Errorf("expected status success, got %q", got.Status)
	}

	if got.StatusCode == nil || *got.StatusCode != statusCode {
		t.Errorf("expected status code %d, got %v", statusCode, got.StatusCode)
	}

	if got.LatencyMs != 125 {
		t.Errorf("expected latency 125, got %d", got.LatencyMs)
	}

	if !got.CheckedAt.Equal(result.CheckedAt) {
		t.Errorf("expected checked_at %v, got %v", result.CheckedAt, got.CheckedAt)
	}
}

func TestProbeResultRepositoryListByMonitorID(t *testing.T) {
	db, monitorRepo := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	monitor := createTestMonitor(t, monitorRepo)

	baseTime := time.Now().UTC().Truncate(time.Microsecond)

	resultsToCreate := []domain.ProbeResult{
		{
			MonitorID: monitor.ID,
			Status:    domain.ProbeStatusSuccess,
			LatencyMs: 100,
			CheckedAt: baseTime.Add(-2 * time.Minute),
		},
		{
			MonitorID: monitor.ID,
			Status:    domain.ProbeStatusFailure,
			LatencyMs: 300,
			CheckedAt: baseTime.Add(-time.Minute),
		},
		{
			MonitorID: monitor.ID,
			Status:    domain.ProbeStatusSuccess,
			LatencyMs: 150,
			CheckedAt: baseTime,
		},
	}

	for _, result := range resultsToCreate {
		if err := repo.Create(context.Background(), result); err != nil {
			t.Fatalf("create probe result: %v", err)
		}
	}

	results, err := repo.ListByMonitorID(
		context.Background(),
		monitor.ID,
		10,
	)
	if err != nil {
		t.Fatalf("list probe results: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if results[i].CheckedAt.After(results[i-1].CheckedAt) {
			t.Fatalf("results are not ordered newest first")
		}
	}

	if results[0].LatencyMs != 150 {
		t.Errorf("expected newest result latency 150, got %d", results[0].LatencyMs)
	}

	if results[2].LatencyMs != 100 {
		t.Errorf("expected oldest result latency 100, got %d", results[2].LatencyMs)
	}
}

func TestProbeResultRepositoryLimit(t *testing.T) {
	db, monitorRepo := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	monitor := createTestMonitor(t, monitorRepo)

	baseTime := time.Now().UTC().Truncate(time.Microsecond)

	for i := 0; i < 5; i++ {
		result := domain.ProbeResult{
			MonitorID: monitor.ID,
			Status:    domain.ProbeStatusSuccess,
			LatencyMs: int64(100 + i),
			CheckedAt: baseTime.Add(time.Duration(i) * time.Second),
		}

		if err := repo.Create(context.Background(), result); err != nil {
			t.Fatalf("create probe result: %v", err)
		}
	}

	results, err := repo.ListByMonitorID(
		context.Background(),
		monitor.ID,
		2,
	)
	if err != nil {
		t.Fatalf("list probe results: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].LatencyMs != 104 {
		t.Errorf("expected newest result latency 104, got %d", results[0].LatencyMs)
	}

	if results[1].LatencyMs != 103 {
		t.Errorf("expected second newest result latency 103, got %d", results[1].LatencyMs)
	}
}

func TestProbeResultRepositoryEmpty(t *testing.T) {
	db, monitorRepo := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	monitor := createTestMonitor(t, monitorRepo)

	results, err := repo.ListByMonitorID(
		context.Background(),
		monitor.ID,
		10,
	)
	if err != nil {
		t.Fatalf("list probe results: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}

	if results == nil {
		t.Fatal("expected empty non-nil slice")
	}
}

func TestProbeResultRepositoryInvalidLimit(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	for _, limit := range []int{0, -1} {
		_, err := repo.ListByMonitorID(
			context.Background(),
			"00000000-0000-0000-0000-000000000000",
			limit,
		)

		if err == nil {
			t.Errorf("expected error for limit %d", limit)
		}
	}
}

func TestProbeResultRepositoryCreateMonitorNotFound(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	repo := NewProbeResultRepository(db)

	result := domain.ProbeResult{
		MonitorID: "00000000-0000-0000-0000-000000000000",
		Status:    domain.ProbeStatusSuccess,
		LatencyMs: 100,
		CheckedAt: time.Now().UTC(),
	}

	err := repo.Create(context.Background(), result)
	if err == nil {
		t.Fatal("expected foreign key violation for nonexistent monitor")
	}
}
