package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
	"github.com/askii12/api-observatory/migrations"
)

func setupIntegrationDB(t *testing.T) (*sql.DB, *MonitorRepository) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := migrations.Run(ctx, db); err != nil {
		db.Close()
		t.Fatalf("run migrations: %v", err)
	}

	// Integration tests use a dedicated test database.
	// Clear previous test data before each test.
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE monitors"); err != nil {
		db.Close()
		t.Fatalf("truncate monitors: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		if _, err := db.ExecContext(
			cleanupCtx,
			"TRUNCATE TABLE monitors",
		); err != nil {
			t.Errorf("cleanup monitors: %v", err)
		}

		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	return db, NewMonitorRepository(db)
}

func TestMonitorRepositoryCRUD(t *testing.T) {
	_, repo := setupIntegrationDB(t)

	ctx := context.Background()

	monitor := &domain.Monitor{
		Name:            "Integration Test API",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutMs:       5000,
		Enabled:         true,
	}

	// CREATE
	if err := repo.Create(ctx, monitor); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	if monitor.ID == "" {
		t.Fatal("expected generated monitor ID")
	}

	if monitor.CreatedAt.IsZero() || monitor.UpdatedAt.IsZero() {
		t.Fatal("expected database timestamps")
	}

	// GET BY ID
	found, err := repo.GetByID(ctx, monitor.ID)
	if err != nil {
		t.Fatalf("get monitor: %v", err)
	}

	if found.Name != monitor.Name {
		t.Errorf("expected name %q, got %q", monitor.Name, found.Name)
	}

	if found.URL != monitor.URL {
		t.Errorf("expected URL %q, got %q", monitor.URL, found.URL)
	}

	// UPDATE
	found.Name = "Updated Integration API"
	found.IntervalSeconds = 120

	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("update monitor: %v", err)
	}

	updated, err := repo.GetByID(ctx, monitor.ID)
	if err != nil {
		t.Fatalf("get updated monitor: %v", err)
	}

	if updated.Name != "Updated Integration API" {
		t.Errorf("unexpected updated name: %q", updated.Name)
	}

	if updated.IntervalSeconds != 120 {
		t.Errorf("expected interval 120, got %d", updated.IntervalSeconds)
	}

	// LIST
	monitors, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list monitors: %v", err)
	}

	if len(monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(monitors))
	}

	// DELETE
	if err := repo.Delete(ctx, monitor.ID); err != nil {
		t.Fatalf("delete monitor: %v", err)
	}

	// VERIFY DELETION
	_, err = repo.GetByID(ctx, monitor.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after deletion, got %v", err)
	}
}

func TestMonitorRepositoryDeleteNotFound(t *testing.T) {
	_, repo := setupIntegrationDB(t)

	err := repo.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
