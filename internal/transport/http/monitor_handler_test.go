package httptransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
)

type mockMonitorRepository struct {
	monitors []domain.Monitor
	err      error
}

func (m *mockMonitorRepository) Create(
	ctx context.Context,
	monitor *domain.Monitor,
) error {
	if m.err != nil {
		return m.err
	}

	now := time.Now()

	monitor.ID = "495535e8-22f5-424b-9d70-dcffff7bf17f"
	monitor.CreatedAt = now
	monitor.UpdatedAt = now
	monitor.NextRunAt = now

	m.monitors = append(m.monitors, *monitor)

	return nil
}

func (m *mockMonitorRepository) List(
	ctx context.Context,
) ([]domain.Monitor, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.monitors, nil
}

func (m *mockMonitorRepository) GetByID(
	ctx context.Context,
	id string,
) (*domain.Monitor, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, monitor := range m.monitors {
		if monitor.ID == id {
			return &monitor, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (m *mockMonitorRepository) Update(
	ctx context.Context,
	monitor *domain.Monitor,
) error {
	if m.err != nil {
		return m.err
	}

	for i := range m.monitors {
		if m.monitors[i].ID == monitor.ID {
			monitor.CreatedAt = m.monitors[i].CreatedAt
			monitor.NextRunAt = m.monitors[i].NextRunAt
			monitor.UpdatedAt = time.Now()

			m.monitors[i] = *monitor
			return nil
		}
	}

	return sql.ErrNoRows
}

func (m *mockMonitorRepository) Delete(
	ctx context.Context,
	id string,
) error {
	if m.err != nil {
		return m.err
	}

	for i, monitor := range m.monitors {
		if monitor.ID == id {
			m.monitors = append(
				m.monitors[:i],
				m.monitors[i+1:]...,
			)
			return nil
		}
	}

	return sql.ErrNoRows
}

func newTestServer(repo domain.MonitorRepository) http.Handler {
	handler := NewMonitorHandler(repo)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/monitors", handler.Create)
	mux.HandleFunc("GET /api/v1/monitors", handler.List)
	mux.HandleFunc("GET /api/v1/monitors/{id}", handler.Get)
	mux.HandleFunc("PUT /api/v1/monitors/{id}", handler.Update)
	mux.HandleFunc("DELETE /api/v1/monitors/{id}", handler.Delete)

	return mux
}

func TestCreateMonitor(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{
		"name": "GitHub",
		"url": "https://github.com",
		"interval_seconds": 60,
		"timeout_ms": 5000,
		"enabled": true
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	var monitor domain.Monitor

	if err := json.NewDecoder(rec.Body).Decode(&monitor); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if monitor.Name != "GitHub" {
		t.Errorf("expected name GitHub, got %s", monitor.Name)
	}

	if monitor.ID == "" {
		t.Error("expected monitor ID to be generated")
	}

	if len(repo.monitors) != 1 {
		t.Errorf(
			"expected 1 monitor in repository, got %d",
			len(repo.monitors),
		)
	}
}

func TestListMonitors(t *testing.T) {
	repo := &mockMonitorRepository{
		monitors: []domain.Monitor{
			{
				ID:   "495535e8-22f5-424b-9d70-dcffff7bf17f",
				Name: "GitHub",
				URL:  "https://github.com",
			},
		},
	}

	server := newTestServer(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/monitors",
		nil,
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	var monitors []domain.Monitor

	if err := json.NewDecoder(rec.Body).Decode(&monitors); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(monitors) != 1 {
		t.Errorf("expected 1 monitor, got %d", len(monitors))
	}
}

func TestGetMonitorNotFound(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/monitors/00000000-0000-0000-0000-000000000000",
		nil,
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestCreateMonitorInvalidURL(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{
		"name": "Invalid",
		"url": "not-a-valid-url",
		"interval_seconds": 60,
		"timeout_ms": 5000,
		"enabled": true
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if len(repo.monitors) != 0 {
		t.Error("invalid monitor must not be saved")
	}
}

func TestDeleteMonitor(t *testing.T) {
	repo := &mockMonitorRepository{
		monitors: []domain.Monitor{
			{
				ID:   "495535e8-22f5-424b-9d70-dcffff7bf17f",
				Name: "GitHub",
			},
		},
	}

	server := newTestServer(repo)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/monitors/495535e8-22f5-424b-9d70-dcffff7bf17f",
		nil,
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if len(repo.monitors) != 0 {
		t.Error("monitor should have been deleted")
	}
}

func TestCreateMonitorRepositoryError(t *testing.T) {
	repo := &mockMonitorRepository{
		err: errors.New("database unavailable"),
	}

	server := newTestServer(repo)

	body := `{
		"name": "GitHub",
		"url": "https://github.com"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestUpdateMonitor(t *testing.T) {
	repo := &mockMonitorRepository{
		monitors: []domain.Monitor{
			{
				ID:              "495535e8-22f5-424b-9d70-dcffff7bf17f",
				Name:            "GitHub",
				URL:             "https://github.com",
				IntervalSeconds: 60,
				TimeoutMs:       5000,
				Enabled:         true,
			},
		},
	}

	server := newTestServer(repo)

	body := `{
		"name": "GitHub Updated",
		"url": "https://github.com",
		"interval_seconds": 120,
		"timeout_ms": 7000,
		"enabled": true
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/monitors/495535e8-22f5-424b-9d70-dcffff7bf17f",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var updated domain.Monitor

	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if updated.Name != "GitHub Updated" {
		t.Errorf("unexpected name: %s", updated.Name)
	}

	if updated.IntervalSeconds != 120 {
		t.Errorf("unexpected interval: %d", updated.IntervalSeconds)
	}
}

func TestUpdateMonitorNotFound(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{
		"name": "Missing",
		"url": "https://example.com",
		"interval_seconds": 60,
		"timeout_ms": 5000,
		"enabled": true
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/monitors/495535e8-22f5-424b-9d70-dcffff7bf17f",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestCreateMonitorMalformedJSON(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(`{"name": "GitHub"`),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateMonitorUnknownField(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{
		"name": "GitHub",
		"url": "https://github.com",
		"unexpected_field": "value"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestCreateMonitorBodyTooLarge(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{"name":"` + strings.Repeat("a", 1<<20) + `","url":"https://example.com"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateMonitorMultipleJSONObjects(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	body := `{"name":"First","url":"https://example.com"} {"name":"Second","url":"https://example.org"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/monitors",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetMonitorInvalidUUID(t *testing.T) {
	repo := &mockMonitorRepository{}
	server := newTestServer(repo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/monitors/not-a-uuid",
		nil,
	)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateMonitorInvalidInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
	}{
		{name: "below minimum", interval: 29},
		{name: "above maximum", interval: 86401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMonitorRepository{}
			server := newTestServer(repo)

			body := fmt.Sprintf(
				`{"name":"Test API","url":"https://example.com","interval_seconds":%d}`,
				tt.interval,
			)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/monitors",
				strings.NewReader(body),
			)

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestCreateMonitorInvalidTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
	}{
		{name: "below minimum", timeout: 499},
		{name: "above maximum", timeout: 30001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMonitorRepository{}
			server := newTestServer(repo)

			body := fmt.Sprintf(
				`{"name":"Test API","url":"https://example.com","timeout_ms":%d}`,
				tt.timeout,
			)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/monitors",
				strings.NewReader(body),
			)

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}
func (m *mockMonitorRepository) ClaimDue(
	ctx context.Context,
	workerID string,
	limit int,
	leaseDuration time.Duration,
) ([]domain.Monitor, error) {
	return []domain.Monitor{}, nil
}
