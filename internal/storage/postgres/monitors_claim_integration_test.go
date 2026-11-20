package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
)

func createClaimTestMonitor(
	t *testing.T,
	repo *MonitorRepository,
	name string,
	enabled bool,
) domain.Monitor {
	t.Helper()

	monitor := domain.Monitor{
		Name:            name,
		URL:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutMs:       5000,
		Enabled:         enabled,
	}

	if err := repo.Create(context.Background(), &monitor); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	return monitor
}

func TestMonitorRepositoryClaimDue(t *testing.T) {
	db, repo := setupIntegrationDB(t)
	ctx := context.Background()

	createClaimTestMonitor(t, repo, "Monitor 1", true)
	createClaimTestMonitor(t, repo, "Monitor 2", true)
	createClaimTestMonitor(t, repo, "Monitor 3", true)

	firstBatch, err := repo.ClaimDue(
		ctx,
		"worker-1",
		2,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if len(firstBatch) != 2 {
		t.Fatalf("expected 2 claimed monitors, got %d", len(firstBatch))
	}

	secondBatch, err := repo.ClaimDue(
		ctx,
		"worker-2",
		2,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}

	if len(secondBatch) != 1 {
		t.Fatalf("expected 1 remaining monitor, got %d", len(secondBatch))
	}

	thirdBatch, err := repo.ClaimDue(
		ctx,
		"worker-3",
		2,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("third claim: %v", err)
	}

	if len(thirdBatch) != 0 {
		t.Fatalf("expected 0 monitors, got %d", len(thirdBatch))
	}

	// Verify that all claimed monitors have an active lease.
	var leasedCount int

	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM monitors
		 WHERE lease_owner IS NOT NULL
		   AND lease_expires_at > NOW()`,
	).Scan(&leasedCount)
	if err != nil {
		t.Fatalf("count leased monitors: %v", err)
	}

	if leasedCount != 3 {
		t.Errorf("expected 3 leased monitors, got %d", leasedCount)
	}
}

func TestMonitorRepositoryClaimDueReclaimsExpiredLease(t *testing.T) {
	db, repo := setupIntegrationDB(t)
	ctx := context.Background()

	monitor := createClaimTestMonitor(t, repo, "Expiring monitor", true)

	firstBatch, err := repo.ClaimDue(
		ctx,
		"worker-1",
		1,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if len(firstBatch) != 1 {
		t.Fatalf("expected 1 claimed monitor, got %d", len(firstBatch))
	}

	// Simulate an expired lease without waiting.
	_, err = db.ExecContext(
		ctx,
		`UPDATE monitors
		 SET lease_expires_at = NOW() - INTERVAL '1 second'
		 WHERE id = $1`,
		monitor.ID,
	)
	if err != nil {
		t.Fatalf("expire lease: %v", err)
	}

	secondBatch, err := repo.ClaimDue(
		ctx,
		"worker-2",
		1,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}

	if len(secondBatch) != 1 {
		t.Fatalf("expected expired lease to be reclaimed, got %d monitors", len(secondBatch))
	}

	if secondBatch[0].ID != monitor.ID {
		t.Errorf(
			"expected monitor %s to be reclaimed, got %s",
			monitor.ID,
			secondBatch[0].ID,
		)
	}
}

func TestMonitorRepositoryClaimDueSkipsDisabledAndFutureMonitors(t *testing.T) {
	db, repo := setupIntegrationDB(t)
	ctx := context.Background()

	createClaimTestMonitor(t, repo, "Disabled monitor", false)

	futureMonitor := createClaimTestMonitor(
		t,
		repo,
		"Future monitor",
		true,
	)

	_, err := db.ExecContext(
		ctx,
		`UPDATE monitors
		 SET next_run_at = NOW() + INTERVAL '1 hour'
		 WHERE id = $1`,
		futureMonitor.ID,
	)
	if err != nil {
		t.Fatalf("schedule future monitor: %v", err)
	}

	claimed, err := repo.ClaimDue(
		ctx,
		"worker-1",
		10,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("claim due monitors: %v", err)
	}

	if len(claimed) != 0 {
		t.Fatalf("expected no monitors to be claimed, got %d", len(claimed))
	}
}

func TestMonitorRepositoryClaimDueRejectsInvalidArguments(t *testing.T) {
	_, repo := setupIntegrationDB(t)
	ctx := context.Background()

	tests := []struct {
		name          string
		workerID      string
		limit         int
		leaseDuration time.Duration
	}{
		{
			name:          "empty worker ID",
			workerID:      "",
			limit:         1,
			leaseDuration: time.Minute,
		},
		{
			name:          "whitespace worker ID",
			workerID:      "   ",
			limit:         1,
			leaseDuration: time.Minute,
		},
		{
			name:          "zero limit",
			workerID:      "worker-1",
			limit:         0,
			leaseDuration: time.Minute,
		},
		{
			name:          "negative limit",
			workerID:      "worker-1",
			limit:         -1,
			leaseDuration: time.Minute,
		},
		{
			name:          "zero lease duration",
			workerID:      "worker-1",
			limit:         1,
			leaseDuration: 0,
		},
		{
			name:          "negative lease duration",
			workerID:      "worker-1",
			limit:         1,
			leaseDuration: -time.Second,
		},
		{
			name:          "sub-microsecond lease duration",
			workerID:      "worker-1",
			limit:         1,
			leaseDuration: time.Nanosecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.ClaimDue(
				ctx,
				tt.workerID,
				tt.limit,
				tt.leaseDuration,
			)

			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}
