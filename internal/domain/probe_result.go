package domain

import "time"

type ProbeStatus string

const (
	ProbeStatusSuccess ProbeStatus = "success"
	ProbeStatusFailure ProbeStatus = "failure"
)

type ProbeResult struct {
	ID           string      `json:"id"`
	MonitorID    string      `json:"monitor_id"`
	Status       ProbeStatus `json:"status"`
	StatusCode   *int        `json:"status_code,omitempty"`
	LatencyMs    int64       `json:"latency_ms"`
	ErrorMessage *string     `json:"error_message,omitempty"`
	CheckedAt    time.Time   `json:"checked_at"`
}
