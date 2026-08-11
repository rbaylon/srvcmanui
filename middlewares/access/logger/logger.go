package accesslogger

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		fmt.Println(r.Method, r.URL.Path, strings.TrimSpace(r.Header.Get("X-Real-IP")), strings.TrimSpace(r.Header.Get("X-Forwarded-For")), ip)
		next.ServeHTTP(w, r)
	})
}
