package loginguard

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP is the address to rate-limit a request under.
//
// This app runs behind an nginx reverse proxy, so the socket peer is the
// proxy and the real client is in X-Real-IP. That header is only believed
// when the peer is loopback or private (the proxy); from any other peer it
// is client-supplied and would let an attacker pick a fresh "address" per
// attempt. IPv6 clients are keyed by their /64, since one subscriber
// controls a whole /64 and could otherwise rotate through it.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	addr := peer
	if peer.IsLoopback() || peer.IsPrivate() {
		if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
			if a, err := netip.ParseAddr(v); err == nil {
				addr = a.Unmap()
			}
		}
	}
	if addr.Is6() {
		return netip.PrefixFrom(addr, 64).Masked().String()
	}
	return addr.String()
}
