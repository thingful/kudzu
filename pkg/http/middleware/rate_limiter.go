package middleware

import (
	"golang.org/x/time/rate"
)

type RateLimiterMiddleware struct {
}

func NewRateLimiterMiddleware(r rate.Limit, b int) *RateLimiterMiddleware {

}
