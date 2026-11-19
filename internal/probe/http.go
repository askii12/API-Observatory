package probe

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/askii12/api-observatory/internal/domain"
)

const maxResponseBodyBytes = 64 * 1024

type Engine interface {
	Check(ctx context.Context, monitor domain.Monitor) domain.ProbeResult
}

type HTTPProbe struct {
	resolver    IPResolver
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func NewHTTPProbe(resolver IPResolver) *HTTPProbe {
	if resolver == nil {
		resolver = defaultResolver{}
	}

	dialer := &net.Dialer{}

	return &HTTPProbe{
		resolver: resolver,
		dialContext: func(
			ctx context.Context,
			network string,
			address string,
		) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
	}
}

func (p *HTTPProbe) Check(
	ctx context.Context,
	monitor domain.Monitor,
) domain.ProbeResult {
	startedAt := time.Now()

	result := domain.ProbeResult{
		MonitorID: monitor.ID,
		Status:    domain.ProbeStatusFailure,
		CheckedAt: startedAt.UTC(),
	}

	if monitor.TimeoutMs <= 0 {
		return failedResult(result, "monitor timeout must be positive")
	}

	checkCtx, cancel := context.WithTimeout(
		ctx,
		time.Duration(monitor.TimeoutMs)*time.Millisecond,
	)
	defer cancel()
	parsedURL, addresses, err := validateTarget(
		checkCtx,
		monitor.URL,
		p.resolver,
	)
	if err != nil {
		result.LatencyMs = time.Since(startedAt).Milliseconds()
		result.CheckedAt = time.Now().UTC()
		return failedResult(result, err.Error())
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		TLSHandshakeTimeout:   time.Duration(monitor.TimeoutMs) * time.Millisecond,
		ResponseHeaderTimeout: time.Duration(monitor.TimeoutMs) * time.Millisecond,
		DialContext: func(
			ctx context.Context,
			network string,
			address string,
		) (net.Conn, error) {
			return p.dialValidatedIP(
				ctx,
				network,
				address,
				parsedURL,
				addresses,
			)
		},
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}

	defer transport.CloseIdleConnections()

	request, err := http.NewRequestWithContext(
		checkCtx,
		http.MethodGet,
		parsedURL.String(),
		nil,
	)
	if err != nil {
		return failedResult(result, fmt.Sprintf("create HTTP request: %v", err))
	}

	request.Header.Set("User-Agent", "API-Observatory/1.0")
	request.Header.Set("Accept-Encoding", "identity")

	response, err := client.Do(request)
	if err != nil {
		result.LatencyMs = time.Since(startedAt).Milliseconds()
		result.CheckedAt = time.Now().UTC()

		return failedResult(
			result,
			fmt.Sprintf("HTTP request failed: %v", err),
		)
	}
	result.LatencyMs = time.Since(startedAt).Milliseconds()
	result.CheckedAt = time.Now().UTC()
	defer response.Body.Close()

	// Read at most 64 KiB + 1 byte. The body is not needed for
	// availability checks, but bounded reading avoids unbounded downloads.
	_, _ = io.Copy(
		io.Discard,
		io.LimitReader(response.Body, maxResponseBodyBytes+1),
	)

	if err := checkCtx.Err(); err != nil {
		result.CheckedAt = time.Now().UTC()
		return failedResult(
			result,
			fmt.Sprintf("HTTP probe timed out or was cancelled: %v", err),
		)
	}

	statusCode := response.StatusCode
	result.StatusCode = &statusCode

	if statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
		result.Status = domain.ProbeStatusSuccess
		return result
	}

	return failedResult(
		result,
		fmt.Sprintf("unexpected HTTP status code: %d", statusCode),
	)
}

func (p *HTTPProbe) dialValidatedIP(
	ctx context.Context,
	network string,
	address string,
	targetURL *url.URL,
	addresses []netip.Addr,
) (net.Conn, error) {
	requestedHost, requestedPort, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address: %w", err)
	}

	targetHost := strings.TrimSuffix(
		strings.ToLower(targetURL.Hostname()),
		".",
	)

	if !strings.EqualFold(
		strings.TrimSuffix(strings.ToLower(requestedHost), "."),
		targetHost,
	) {
		return nil, fmt.Errorf("unexpected dial hostname")
	}

	expectedPort := targetURL.Port()
	if expectedPort == "" {
		if targetURL.Scheme == "https" {
			expectedPort = "443"
		} else {
			expectedPort = "80"
		}
	}

	if requestedPort != expectedPort {
		return nil, fmt.Errorf("unexpected dial port")
	}

	var lastErr error

	for _, ip := range addresses {
		if !isPublicAddress(ip) {
			return nil, fmt.Errorf("refusing to dial a non-public IP address")
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		dialAddress := net.JoinHostPort(ip.String(), requestedPort)

		conn, err := p.dialContext(ctx, network, dialAddress)
		if err == nil {
			return conn, nil
		}

		lastErr = err
	}

	if lastErr == nil {
		return nil, fmt.Errorf("no validated IP addresses available")
	}

	return nil, fmt.Errorf("could not connect to validated IP addresses: %w", lastErr)
}

func failedResult(
	result domain.ProbeResult,
	message string,
) domain.ProbeResult {
	result.Status = domain.ProbeStatusFailure
	result.ErrorMessage = &message
	return result
}
