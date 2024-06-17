package middleware

import (
	"errors"
	"net/http"

	"github.com/yamauthi/goexpert-rate-limiter/internal/limiter"
)

type LimiterMiddleware struct {
	Limiter *limiter.Limiter
}

func NewLimiterMiddleware(limiter *limiter.Limiter) *LimiterMiddleware {
	return &LimiterMiddleware{
		Limiter: limiter,
	}
}

func (m *LimiterMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("API_KEY")
		clientIP := GetIP(r)
		allowed, err := m.Limiter.AllowRequest(clientIP, apiKey)
		if allowed {
			next.ServeHTTP(w, r)
			return
		}

		if errors.Is(err, limiter.ErrMaxNumberRequestsReached) {
			http.Error(
				w,
				err.Error(),
				http.StatusTooManyRequests,
			)
			return
		}

		if errors.Is(err, limiter.ErrInvalidClient) ||
			errors.Is(err, limiter.ErrApiKeyNotFound) {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
	})
}

func GetIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
