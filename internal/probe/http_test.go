package probe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
)

func newTestProbe(
	t *testing.T,
	handler http.Handler,
) (*HTTPProbe, string, *atomic.Int32, *atomic.Value) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	targetURL := fmt.Sprintf(
		"http://example.com:%s/test",
		serverURL.Port(),
	)

	resolver := fakeResolver{
		addresses: []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
		},
	}

	var dialCount atomic.Int32
	var dialAddress atomic.Value

	probe := NewHTTPProbe(resolver)
	probe.dialContext = func(
		ctx context.Context,
		network string,
		address string,
	) (net.Conn, error) {
		dialCount.Add(1)
		dialAddress.Store(address)

		dialer := &net.Dialer{}
		return dialer.DialContext(
			ctx,
			network,
			server.Listener.Addr().String(),
		)
	}

	return probe, targetURL, &dialCount, &dialAddress
}

func TestHTTPProbeStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus domain.ProbeStatus
	}{
		{
			name:       "HTTP 200",
			statusCode: http.StatusOK,
			wantStatus: domain.ProbeStatusSuccess,
		},
		{
			name:       "HTTP 204",
			statusCode: http.StatusNoContent,
			wantStatus: domain.ProbeStatusSuccess,
		},
		{
			name:       "HTTP 301",
			statusCode: http.StatusMovedPermanently,
			wantStatus: domain.ProbeStatusSuccess,
		},
		{
			name:       "HTTP 404",
			statusCode: http.StatusNotFound,
			wantStatus: domain.ProbeStatusFailure,
		},
		{
			name:       "HTTP 500",
			statusCode: http.StatusInternalServerError,
			wantStatus: domain.ProbeStatusFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.WriteHeader(tt.statusCode)
			})

			probe, targetURL, _, _ := newTestProbe(t, handler)

			result := probe.Check(
				context.Background(),
				domain.Monitor{
					ID:        "test-monitor",
					URL:       targetURL,
					TimeoutMs: 1000,
				},
			)

			if result.Status != tt.wantStatus {
				t.Fatalf(
					"status = %q, want %q; error: %v",
					result.Status,
					tt.wantStatus,
					result.ErrorMessage,
				)
			}

			if result.StatusCode == nil {
				t.Fatal("expected HTTP status code")
			}

			if *result.StatusCode != tt.statusCode {
				t.Fatalf(
					"status code = %d, want %d",
					*result.StatusCode,
					tt.statusCode,
				)
			}

			if result.LatencyMs < 0 {
				t.Fatalf("latency cannot be negative: %d", result.LatencyMs)
			}

			if result.CheckedAt.IsZero() {
				t.Fatal("expected CheckedAt timestamp")
			}
		})
	}
}

func TestHTTPProbeUsesValidatedIP(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	probe, targetURL, _, dialAddress := newTestProbe(t, handler)

	result := probe.Check(
		context.Background(),
		domain.Monitor{
			ID:        "test-monitor",
			URL:       targetURL,
			TimeoutMs: 1000,
		},
	)

	if result.Status != domain.ProbeStatusSuccess {
		t.Fatalf("expected success, got %q: %v", result.Status, result.ErrorMessage)
	}

	got := dialAddress.Load()
	if got == nil {
		t.Fatal("expected dialer to be called")
	}

	expected := "93.184.216.34:" + mustPort(t, targetURL)

	if got.(string) != expected {
		t.Fatalf("dial address = %q, want %q", got, expected)
	}
}

func TestHTTPProbeDoesNotFollowRedirect(t *testing.T) {
	var requestCount atomic.Int32

	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		requestCount.Add(1)

		w.Header().Set(
			"Location",
			"http://127.0.0.1:12345/internal",
		)
		w.WriteHeader(http.StatusFound)
	})

	probe, targetURL, dialCount, _ := newTestProbe(t, handler)

	result := probe.Check(
		context.Background(),
		domain.Monitor{
			ID:        "test-monitor",
			URL:       targetURL,
			TimeoutMs: 1000,
		},
	)

	if result.Status != domain.ProbeStatusSuccess {
		t.Fatalf("expected redirect response to count as success, got %q", result.Status)
	}

	if result.StatusCode == nil || *result.StatusCode != http.StatusFound {
		t.Fatalf("expected HTTP 302, got %v", result.StatusCode)
	}

	if requestCount.Load() != 1 {
		t.Fatalf("server received %d requests, want 1", requestCount.Load())
	}

	if dialCount.Load() != 1 {
		t.Fatalf("dialer called %d times, want 1", dialCount.Load())
	}
}

func TestHTTPProbeRejectsPrivateDNSResult(t *testing.T) {
	var dialCount atomic.Int32

	probe := NewHTTPProbe(fakeResolver{
		addresses: []netip.Addr{
			netip.MustParseAddr("127.0.0.1"),
		},
	})

	probe.dialContext = func(
		ctx context.Context,
		network string,
		address string,
	) (net.Conn, error) {
		dialCount.Add(1)
		return nil, fmt.Errorf("should not be called")
	}

	result := probe.Check(
		context.Background(),
		domain.Monitor{
			ID:        "test-monitor",
			URL:       "http://example.com",
			TimeoutMs: 1000,
		},
	)

	if result.Status != domain.ProbeStatusFailure {
		t.Fatalf("expected failure, got %q", result.Status)
	}

	if dialCount.Load() != 0 {
		t.Fatal("dialer must not be called for a private DNS result")
	}

	if result.ErrorMessage == nil {
		t.Fatal("expected error message")
	}
}

func TestHTTPProbeTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	probe, targetURL, _, _ := newTestProbe(t, handler)

	result := probe.Check(
		context.Background(),
		domain.Monitor{
			ID:        "test-monitor",
			URL:       targetURL,
			TimeoutMs: 30,
		},
	)

	if result.Status != domain.ProbeStatusFailure {
		t.Fatalf("expected timeout failure, got %q", result.Status)
	}

	if result.ErrorMessage == nil {
		t.Fatal("expected timeout error message")
	}
}

func TestHTTPProbeRejectsInvalidTimeout(t *testing.T) {
	probe := NewHTTPProbe(nil)

	result := probe.Check(
		context.Background(),
		domain.Monitor{
			ID:        "test-monitor",
			URL:       "https://example.com",
			TimeoutMs: 0,
		},
	)

	if result.Status != domain.ProbeStatusFailure {
		t.Fatalf("expected failure, got %q", result.Status)
	}

	if result.ErrorMessage == nil {
		t.Fatal("expected error message")
	}
}

func mustPort(t *testing.T, rawURL string) string {
	t.Helper()

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	port := parsedURL.Port()
	if port == "" {
		t.Fatal("expected test URL to contain a port")
	}

	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("invalid test port: %v", err)
	}

	return port
}
