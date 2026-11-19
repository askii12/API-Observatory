package httptransport

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/askii12/api-observatory/internal/domain"
)

type MonitorHandler struct {
	repo domain.MonitorRepository
}

func NewMonitorHandler(repo domain.MonitorRepository) *MonitorHandler {
	return &MonitorHandler{repo: repo}
}

type createMonitorRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds *int   `json:"interval_seconds"`
	TimeoutMs       *int   `json:"timeout_ms"`
	Enabled         *bool  `json:"enabled"`
}

type updateMonitorRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutMs       int    `json:"timeout_ms"`
	Enabled         *bool  `json:"enabled"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func decodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain only one JSON object")
	}

	return nil
}

func validURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Host != ""
}

func validMonitorInput(name, targetURL string, interval, timeout int) bool {
	return strings.TrimSpace(name) != "" &&
		len(name) <= 120 &&
		validURL(targetURL) &&
		interval >= 30 && interval <= 86400 &&
		timeout >= 500 && timeout <= 30000
}

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createMonitorRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	interval := 60
	if req.IntervalSeconds != nil {
		interval = *req.IntervalSeconds
	}

	timeout := 5000
	if req.TimeoutMs != nil {
		timeout = *req.TimeoutMs
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if !validMonitorInput(req.Name, req.URL, interval, timeout) {
		writeError(w, http.StatusBadRequest, "invalid monitor fields")
		return
	}

	monitor := &domain.Monitor{
		Name:            strings.TrimSpace(req.Name),
		URL:             req.URL,
		IntervalSeconds: interval,
		TimeoutMs:       timeout,
		Enabled:         enabled,
	}

	if err := h.repo.Create(r.Context(), monitor); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}

	writeJSON(w, http.StatusCreated, monitor)
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitors")
		return
	}

	writeJSON(w, http.StatusOK, monitors)
}

func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid monitor ID")
		return
	}

	monitor, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get monitor")
		return
	}

	writeJSON(w, http.StatusOK, monitor)
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid monitor ID")
		return
	}

	var req updateMonitorRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Enabled == nil ||
		!validMonitorInput(req.Name, req.URL, req.IntervalSeconds, req.TimeoutMs) {
		writeError(w, http.StatusBadRequest, "invalid monitor fields")
		return
	}

	monitor := &domain.Monitor{
		ID:              id,
		Name:            strings.TrimSpace(req.Name),
		URL:             req.URL,
		IntervalSeconds: req.IntervalSeconds,
		TimeoutMs:       req.TimeoutMs,
		Enabled:         *req.Enabled,
	}

	err := h.repo.Update(r.Context(), monitor)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}

	writeJSON(w, http.StatusOK, monitor)
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid monitor ID")
		return
	}

	err := h.repo.Delete(r.Context(), id)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
