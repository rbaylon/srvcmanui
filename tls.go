package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/rbaylon/srvcman/database"
)

// fqdnPattern matches a syntactically valid public DNS hostname with at
// least one dot (e.g. "ui.example.com") - not an IP literal and not a
// single-label name like "localhost", neither of which Let's Encrypt can
// ever issue a certificate for.
var fqdnPattern = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}$`)

// isValidFQDN reports whether s looks like a public domain name ACME could
// actually issue a certificate for.
func isValidFQDN(s string) bool {
	if s == "" || net.ParseIP(s) != nil {
		return false
	}
	return fqdnPattern.MatchString(s)
}

// serveTLS runs handler over HTTPS - this app no longer serves plain HTTP.
// With a valid public domain in APP_DOMAIN, certificates are obtained and
// renewed automatically from Let's Encrypt via certmagic; otherwise (no
// domain configured, or APP_DOMAIN isn't a real public FQDN - an IP
// address, "localhost", a bare LAN hostname) a self-signed certificate is
// generated and cached on disk, so it doesn't change - and doesn't need
// re-trusting in the browser - across restarts.
//
// The two paths use different ports by default, and that's not a style
// choice: Let's Encrypt's HTTP-01/TLS-ALPN-01 challenges are only ever
// validated against the requester's real port 80/443, so certmagic binds
// those rather than APP_PORT. Binding 80/443 needs root, which this app
// otherwise has no reason to run as - see APP_TLS_HTTP_PORT/
// APP_TLS_HTTPS_PORT on serveACME for how to avoid that. The self-signed
// path has no such constraint and keeps using APP_PORT, same port this
// app has always listened on, which is never privileged in practice.
func serveTLS(handler http.Handler, port string) error {
	domain := strings.TrimSpace(database.GetEnvVariable("APP_DOMAIN"))
	if isValidFQDN(domain) {
		return serveACME(handler, domain)
	}
	if domain != "" {
		log.Printf("TLS: APP_DOMAIN=%q is not a valid public domain name (an IP address or bare hostname can't get a Let's Encrypt certificate) - falling back to a self-signed certificate", domain)
	}
	return serveSelfSigned(handler, port)
}

// serveACME obtains/renews a Let's Encrypt certificate for domain and
// serves handler over it. By default this binds the real ports 80
// (HTTP-01 challenges plus a redirect to https) and 443, which needs
// root.
//
// APP_TLS_HTTP_PORT/APP_TLS_HTTPS_PORT override that, so this process can
// stay an ordinary user: point them at unprivileged ports a firewall rule
// redirects the real 80/443 to (e.g. OpenBSD pf's `rdr-to`), and this
// process only ever binds those. Nothing changes from Let's Encrypt's
// side - it still connects to the real port 80/443 from outside; the
// firewall remaps the packet before it reaches this process's socket, so
// the ACME challenge validates exactly the same either way. certmagic
// supports this directly (see its HTTPPort/HTTPSPort vars) - it's not a
// workaround bolted on here.
func serveACME(handler http.Handler, domain string) error {
	if email := strings.TrimSpace(database.GetEnvVariable("APP_TLS_EMAIL")); email != "" {
		certmagic.DefaultACME.Email = email
	}
	certmagic.DefaultACME.Agreed = true

	httpPort, httpsPort := applyTLSPortOverrides()
	log.Printf("TLS: requesting/renewing a Let's Encrypt certificate for %s via certmagic (binding ports %d and %d)", domain, httpPort, httpsPort)
	return certmagic.HTTPS([]string{domain}, handler)
}

// applyTLSPortOverrides reads APP_TLS_HTTP_PORT/APP_TLS_HTTPS_PORT and
// writes them into certmagic's own HTTPPort/HTTPSPort vars (falling back
// to certmagic's existing defaults, ordinarily 80/443, when unset).
// Split out from serveACME so this side effect can be tested on its own,
// synchronously, without also triggering certmagic.HTTPS's real network/
// ACME calls.
func applyTLSPortOverrides() (httpPort, httpsPort int) {
	httpPort = tlsPortOrDefault("APP_TLS_HTTP_PORT", certmagic.HTTPPort)
	httpsPort = tlsPortOrDefault("APP_TLS_HTTPS_PORT", certmagic.HTTPSPort)
	certmagic.HTTPPort = httpPort
	certmagic.HTTPSPort = httpsPort
	return httpPort, httpsPort
}

// tlsPortOrDefault reads a port number from the named env var, falling
// back to def if it's unset, empty, or not a valid port number.
func tlsPortOrDefault(envVar string, def int) int {
	v := strings.TrimSpace(database.GetEnvVariable(envVar))
	if v == "" {
		return def
	}
	port, err := strconv.Atoi(v)
	if err != nil || port < 1 || port > 65535 {
		log.Printf("TLS: %s=%q is not a valid port number, using %d", envVar, v, def)
		return def
	}
	return port
}

// serveSelfSigned serves handler over HTTPS on APP_PORT using a
// self-signed certificate, generating and caching one under ./tls/ if none
// is cached yet or the cached one is expired/expiring soon.
func serveSelfSigned(handler http.Handler, port string) error {
	cert, err := loadOrCreateSelfSigned()
	if err != nil {
		return fmt.Errorf("preparing self-signed certificate: %w", err)
	}
	srv := &http.Server{
		Addr:      ":" + port,
		Handler:   handler,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	log.Printf("TLS: serving with a self-signed certificate on port %s - browsers will show a one-time trust warning; this is expected for direct IP/LAN access without a public domain", port)
	return srv.ListenAndServeTLS("", "")
}

const (
	selfSignedCertPath = "./tls/selfsigned.crt"
	selfSignedKeyPath  = "./tls/selfsigned.key"
	// selfSignedValidity is kept under browsers' ~398-day maximum
	// certificate lifetime (Safari/Chrome both reject longer-lived leaf
	// certs), so a cached cert can't silently become browser-invalid
	// without also failing the expiry check below that regenerates it.
	selfSignedValidity   = 397 * 24 * time.Hour
	selfSignedRenewAhead = 30 * 24 * time.Hour
)

// loadOrCreateSelfSigned returns the cached self-signed certificate,
// generating a fresh one if none is cached or the cached one is expired or
// expiring within selfSignedRenewAhead.
func loadOrCreateSelfSigned() (tls.Certificate, error) {
	if cert, err := tls.LoadX509KeyPair(selfSignedCertPath, selfSignedKeyPath); err == nil {
		if leaf, perr := x509.ParseCertificate(cert.Certificate[0]); perr == nil && time.Now().Before(leaf.NotAfter.Add(-selfSignedRenewAhead)) {
			log.Printf("TLS: reusing cached self-signed certificate (valid until %s)", leaf.NotAfter.Format("2006-01-02"))
			return cert, nil
		}
		log.Println("TLS: cached self-signed certificate missing, unreadable, or expiring soon - regenerating")
	}
	return generateSelfSigned()
}

// generateSelfSigned creates a new self-signed ECDSA certificate covering
// localhost and this host's own non-loopback IPs, writes it to disk, and
// returns it loaded as a tls.Certificate.
func generateSelfSigned() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial number: %w", err)
	}
	notBefore := time.Now().Add(-time.Hour)
	notAfter := notBefore.Add(selfSignedValidity)
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "srvcmanui (self-signed)"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  selfSignedIPs(),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshaling key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(selfSignedCertPath), 0o755); err != nil {
		return tls.Certificate{}, fmt.Errorf("creating tls/ directory: %w", err)
	}
	if err := os.WriteFile(selfSignedCertPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return tls.Certificate{}, fmt.Errorf("writing certificate: %w", err)
	}
	// Private key file is not world/group readable - anyone who can read
	// it can impersonate this server to any client that trusted the cert.
	if err := os.WriteFile(selfSignedKeyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return tls.Certificate{}, fmt.Errorf("writing key: %w", err)
	}
	log.Printf("TLS: generated a new self-signed certificate, valid until %s", notAfter.Format("2006-01-02"))
	return tls.LoadX509KeyPair(selfSignedCertPath, selfSignedKeyPath)
}

// selfSignedIPs is every IP address a browser might actually be pointed at
// to reach this host directly: loopback, plus every non-loopback address
// this machine currently has configured. A self-signed cert is untrusted
// regardless, but matching the address at least avoids stacking a SAN
// mismatch error on top of that expected warning.
func selfSignedIPs() []net.IP {
	ips := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ips = append(ips, ipNet.IP)
	}
	return ips
}
