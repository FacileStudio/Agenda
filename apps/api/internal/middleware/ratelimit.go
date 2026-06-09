package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"api/internal/errors"
	"api/internal/httpjson"
)

type visitor struct {
	count     int
	windowEnd time.Time
}

func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	var visitors sync.Map

	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			visitors.Range(func(key, value any) bool {
				v := value.(*visitor)
				if now.After(v.windowEnd) {
					visitors.Delete(key)
				}
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			now := time.Now()

			val, _ := visitors.LoadOrStore(ip, &visitor{
				count:     0,
				windowEnd: now.Add(window),
			})
			v := val.(*visitor)

			if now.After(v.windowEnd) {
				v.count = 0
				v.windowEnd = now.Add(window)
			}

			if v.count >= requests {
				retryAfter := int(time.Until(v.windowEnd).Seconds()) + 1
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				httpjson.WriteError(w, errors.RateLimited("too many requests, try again later"))
				return
			}

			v.count++
			next.ServeHTTP(w, r)
		})
	}
}
