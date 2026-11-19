package probe

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

type fakeResolver struct {
	addresses []netip.Addr
	err       error
}

func (f fakeResolver) LookupNetIP(
	ctx context.Context,
	network string,
	host string,
) ([]netip.Addr, error) {
	return f.addresses, f.err
}

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		resolver  IPResolver
		wantErr   bool
	}{
		{
			name:      "public IPv4",
			targetURL: "https://8.8.8.8",
		},
		{
			name:      "public IPv6",
			targetURL: "https://[2606:4700:4700::1111]",
		},
		{
			name:      "localhost",
			targetURL: "http://localhost",
			wantErr:   true,
		},
		{
			name:      "loopback IPv4",
			targetURL: "http://127.0.0.1",
			wantErr:   true,
		},
		{
			name:      "private IPv4",
			targetURL: "http://192.168.1.1",
			wantErr:   true,
		},
		{
			name:      "link-local IPv4",
			targetURL: "http://169.254.169.254",
			wantErr:   true,
		},
		{
			name:      "private IPv6",
			targetURL: "http://[fc00::1]",
			wantErr:   true,
		},
		{
			name:      "unsupported scheme",
			targetURL: "ftp://example.com",
			wantErr:   true,
		},
		{
			name:      "URL credentials",
			targetURL: "https://user:password@example.com",
			wantErr:   true,
		},
		{
			name:      "invalid port",
			targetURL: "https://example.com:99999",
			wantErr:   true,
		},
		{
			name:      "public DNS result",
			targetURL: "https://example.com",
			resolver: fakeResolver{
				addresses: []netip.Addr{
					netip.MustParseAddr("93.184.216.34"),
				},
			},
		},
		{
			name:      "DNS resolves to private IP",
			targetURL: "https://example.com",
			resolver: fakeResolver{
				addresses: []netip.Addr{
					netip.MustParseAddr("192.168.1.10"),
				},
			},
			wantErr: true,
		},
		{
			name:      "DNS returns mixed public and private IPs",
			targetURL: "https://example.com",
			resolver: fakeResolver{
				addresses: []netip.Addr{
					netip.MustParseAddr("93.184.216.34"),
					netip.MustParseAddr("10.0.0.5"),
				},
			},
			wantErr: true,
		},
		{
			name:      "DNS resolution failure",
			targetURL: "https://example.com",
			resolver: fakeResolver{
				err: errors.New("DNS unavailable"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := validateTarget(
				context.Background(),
				tt.targetURL,
				tt.resolver,
			)

			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsPublicAddress(t *testing.T) {
	tests := []struct {
		address string
		want    bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},
		{"127.0.0.1", false},
		{"169.254.1.1", false},
		{"100.64.0.1", false},
		{"192.0.2.1", false},
		{"198.18.0.1", false},
		{"224.0.0.1", false},
		{"::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
		{"ff02::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			address := netip.MustParseAddr(tt.address)

			got := isPublicAddress(address)
			if got != tt.want {
				t.Errorf(
					"isPublicAddress(%s) = %v, want %v",
					tt.address,
					got,
					tt.want,
				)
			}
		})
	}
}
