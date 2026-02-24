package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	guestLimit  = 30
	guestWindow = time.Minute
)

type bucket struct {
	count int
	start time.Time
}

var (
	mu      sync.Mutex
	buckets = map[string]*bucket{}
)

// Middleware 仅对 context 中 email 以 "guest:" 开头的请求按 IP 限流，超限返回 429。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, _ := r.Context().Value("email").(string)
		if !hasGuestPrefix(email) {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		mu.Lock()
		b := buckets[ip]
		now := time.Now()
		if b == nil || now.Sub(b.start) >= guestWindow {
			b = &bucket{count: 1, start: now}
			buckets[ip] = b
		} else {
			b.count++
		}
		n := b.count
		mu.Unlock()
		if n > guestLimit {
			http.Error(w, "请求过于频繁", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hasGuestPrefix(s string) bool {
	return strings.HasPrefix(s, "guest:")
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		if i := strings.Index(x, ","); i >= 0 {
			x = strings.TrimSpace(x[:i])
		} else {
			x = strings.TrimSpace(x)
		}
		if x != "" {
			return x
		}
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}
