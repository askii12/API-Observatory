package probe

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type IPResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type defaultResolver struct{}

func (defaultResolver) LookupNetIP(
	ctx context.Context,
	network string,
	host string,
) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, network, host)
}

var blockedPrefixes = []netip.Prefix{
	// IPv4 private and special-purpose ranges.
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),

	// IPv6 special-purpose ranges.
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func validateTarget(
	ctx context.Context,
	rawURL string,
	resolver IPResolver,
) (*url.URL, []netip.Addr, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse target URL: %w", err)
	}

	if parsedURL.Opaque != "" {
		return nil, nil, fmt.Errorf("opaque URLs are not allowed")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, nil, fmt.Errorf("only HTTP and HTTPS URLs are allowed")
	}

	if parsedURL.Host == "" || parsedURL.Hostname() == "" {
		return nil, nil, fmt.Errorf("target URL must contain a hostname")
	}

	if parsedURL.User != nil {
		return nil, nil, fmt.Errorf("URL credentials are not allowed")
	}

	if parsedURL.Fragment != "" {
		return nil, nil, fmt.Errorf("URL fragments are not allowed")
	}

	if port := parsedURL.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return nil, nil, fmt.Errorf("invalid target port")
		}
	}

	hostname := strings.TrimSuffix(
		strings.ToLower(parsedURL.Hostname()),
		".",
	)

	if hostname == "" {
		return nil, nil, fmt.Errorf("target hostname is empty")
	}

	if strings.Contains(hostname, "%") {
		return nil, nil, fmt.Errorf("IPv6 zones are not allowed")
	}

	if isLocalHostname(hostname) {
		return nil, nil, fmt.Errorf("local hostnames are not allowed")
	}

	if resolver == nil {
		resolver = defaultResolver{}
	}

	var addresses []netip.Addr

	if ip, err := netip.ParseAddr(hostname); err == nil {
		addresses = []netip.Addr{ip}
	} else {
		addresses, err = resolver.LookupNetIP(ctx, "ip", hostname)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve target hostname: %w", err)
		}
	}

	if len(addresses) == 0 {
		return nil, nil, fmt.Errorf("target hostname resolved to no IP addresses")
	}

	// Reject the entire hostname if even one returned address is unsafe.
	for _, address := range addresses {
		if !isPublicAddress(address) {
			return nil, nil, fmt.Errorf(
				"target hostname resolves to a non-public IP address",
			)
		}
	}

	return parsedURL, addresses, nil
}

func isLocalHostname(hostname string) bool {
	return hostname == "localhost" ||
		strings.HasSuffix(hostname, ".localhost") ||
		strings.HasSuffix(hostname, ".local") ||
		strings.HasSuffix(hostname, ".internal")
}

func isPublicAddress(address netip.Addr) bool {
	if !address.IsValid() || address.Zone() != "" {
		return false
	}

	// Avoid IPv4-mapped IPv6 representation ambiguity.
	if address.Is4In6() {
		address = address.Unmap()
	}

	if !address.IsGlobalUnicast() {
		return false
	}

	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}

	return true
}
