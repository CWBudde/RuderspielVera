package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ruderspiel/internal/gameservice"
)

// ipLimiter is a token bucket per client IP. Buckets that have refilled completely carry no
// state worth keeping, so they are evicted on a periodic sweep; memory is bounded by the number
// of clients active within one refill period.
type ipLimiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	now     func() time.Time
	buckets map[string]*bucket
	swept   time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter(perSecond float64, burst int, now func() time.Time) *ipLimiter {
	return &ipLimiter{rate: perSecond, burst: float64(burst), now: now, buckets: map[string]*bucket{}, swept: now()}
}

// fullAfter is how long an untouched bucket takes to refill completely.
func (l *ipLimiter) fullAfter() time.Duration {
	return time.Duration(l.burst / l.rate * float64(time.Second))
}

// allow takes one token for ip; if none is left it reports false and the wait until the next one.
func (l *ipLimiter) allow(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.swept) >= l.fullAfter() {
		l.evict(now)
	}

	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}

	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens < 1 {
		return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	}

	b.tokens--

	return true, 0
}

// evict drops every bucket that would be full by now. Caller holds l.mu.
func (l *ipLimiter) evict(now time.Time) {
	for ip, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, ip)
		}
	}

	l.swept = now
}

// size returns the number of tracked clients (tests).
func (l *ipLimiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.buckets)
}

// rateLimited wraps a handler with the per-IP creation limit (no-op without a limiter).
func (s *Server) rateLimited(next http.HandlerFunc) http.HandlerFunc {
	if s.limiter == nil {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		ok, wait := s.limiter.allow(ip)
		if !ok {
			s.log.Warn("rate limited", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			writeError(w, gameservice.NewError(http.StatusTooManyRequests, "err.rateLimited"))

			return
		}

		next(w, r)
	}
}

// clientIP is the first X-Forwarded-For entry (set by Traefik, which drops client-sent
// forwarding headers from untrusted sources) or else the host part of RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
