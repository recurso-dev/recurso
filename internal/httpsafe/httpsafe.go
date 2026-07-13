// Package httpsafe guards outbound HTTP against SSRF: it rejects URLs that
// resolve to internal/private addresses and provides a dialer control that
// re-checks the actual connect-time IP (defeating DNS rebinding and redirects
// to internal targets).
package httpsafe

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"syscall"
)

// ErrDisallowedURL is returned for a non-http(s) URL or one whose host resolves
// to a loopback/private/link-local/etc. address.
var ErrDisallowedURL = errors.New("url must be an external http(s) endpoint")

// isDisallowed reports whether ip is an SSRF target: loopback (127/8, ::1),
// private (10/8, 172.16/12, 192.168/16, fc00::/7), link-local (169.254/16 incl.
// the 169.254.169.254 cloud-metadata endpoint, fe80::/10), unspecified, or
// multicast.
func isDisallowed(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// ValidateExternalURL rejects non-http(s) schemes and hosts that resolve to a
// disallowed address. It is the create-time check; DialControl re-validates at
// connect time so a host that later rebinds to an internal IP is still blocked.
func ValidateExternalURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrDisallowedURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrDisallowedURL
	}
	host := u.Hostname()
	if host == "" {
		return ErrDisallowedURL
	}
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowed(ip) {
			return ErrDisallowedURL
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ErrDisallowedURL
	}
	for _, ip := range ips {
		if isDisallowed(ip) {
			return ErrDisallowedURL
		}
	}
	return nil
}

// DialControl is a net.Dialer.Control hook that blocks connecting to a
// disallowed IP even if DNS resolved to it after the create-time check.
func DialControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); ip != nil && isDisallowed(ip) {
		return fmt.Errorf("blocked connection to disallowed address %s", host)
	}
	return nil
}
