package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	registry "github.com/thingful/retryable-registry-prometheus"

	"github.com/jonboulle/clockwork"
	"golang.org/x/time/rate"
)

const (
	defaultRate = 4
)

var (
	limited = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "rate_limited_requests",
			Help:      "A counter of rate limited requests",
		}, []string{"path"},
	)
)

func init() {
	registry.MustRegister(limited)
}

// visitor is a type used to hold visit status for a user of the website. We use
// a struct for this as it allows us to keep track of the time of the last
// visit, so allowing old rate limits to be cleaned up.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// expired is a helper function that checks whether a visitor entry should be
// consided expired and so flagged for deletion from our state. Must be passed a
// valid clockwork.Clock as well as an expiry duration.
func (v *visitor) expired(clock clockwork.Clock, expiry time.Duration) bool {
	if clock.Now().Sub(v.lastSeen) > expiry {
		return true
	}
	return false
}

// RateLimiterMiddleware is a middleware that implements simple rate limiting of
// requests using the golang.org/x/time/rate package.
type RateLimiterMiddleware struct {
	expiry   time.Duration
	clock    clockwork.Clock
	visitors map[string]*visitor
	sync.RWMutex
}

// NewRateLimiterMiddleware returns a new middleware instance that has been
// configured to start limiting requests to the API. We limit by the submitted
// API key that is saved to the context.
func NewRateLimiterMiddleware(clock clockwork.Clock) *RateLimiterMiddleware {
	expiry := time.Duration(60) * time.Second

	rm := &RateLimiterMiddleware{
		expiry:   expiry,
		clock:    clock,
		visitors: make(map[string]*visitor),
	}

	// set up a ticker to remove stale entries every `expiry` seconds
	ticker := clock.NewTicker(expiry)
	go func() {
		for range ticker.Chan() {
			rm.cleanupVisitors()
		}
	}()

	return rm
}

// Handler is the middleware handler function
func (rm *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uid, err := uidFromContext(ctx)
		if err != nil {
			invalidTokenError(w, err)
			return
		}

		rate := rateFromContext(ctx)

		limiter := rm.getVisitor(uid, rate)
		if !limiter.Allow() {
			limited.With(
				prometheus.Labels{
					"path": r.URL.Path,
				},
			).Inc()

			tooManyRequestsError(w, fmt.Errorf("API rate limit exceeded, please try again later. Your current limits are no more than %v req/sec", rate))
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// getVisitor attempts to return a rate limiter for the given uid. If an entry
// for the visitor is already present in the map we simply return it, else we
// hand over to `addVisitor` to create a new one.
func (rm *RateLimiterMiddleware) getVisitor(uid string, rate int) *rate.Limiter {
	rm.RLock()
	if v, ok := rm.visitors[uid]; ok {
		v.lastSeen = rm.clock.Now()
		rm.RUnlock()
		return v.limiter
	}
	rm.RUnlock()

	return rm.addVisitor(uid, rate)
}

// addVisitor is an unexported function that attempts to adds a new visitor into
// our map, initializes its limiter and adds a timestamp at which the visitor
// was received. We later use this timestamp to remove old entries from the map.
func (rm *RateLimiterMiddleware) addVisitor(uid string, r int) *rate.Limiter {
	// create new limiter
	limiter := rate.NewLimiter(rate.Limit(r), r*2)

	// add to our map
	rm.Lock()
	defer rm.Unlock()

	rm.visitors[uid] = &visitor{
		limiter:  limiter,
		lastSeen: rm.clock.Now(),
	}

	return rm.visitors[uid].limiter
}

// cleanupVisitors must be called repeatedly in a goroutine, and is responsible
// for removing stale entries from our map, i.e. visitors that haven't been seen
// for the length of our `expiry` duration.
func (rm *RateLimiterMiddleware) cleanupVisitors() {
	rm.Lock()
	defer rm.Unlock()
	for uid, v := range rm.visitors {
		if v.expired(rm.clock, rm.expiry) {
			delete(rm.visitors, uid)
		}
	}
}

// uidFromContext returns the subject key (i.e. uid) from the context returning
// an error if it isn't there
func uidFromContext(ctx context.Context) (string, error) {
	if uid, ok := ctx.Value(subjectKey).(string); ok {
		return uid, nil
	}

	return "", errors.New("Unable to find subject uid in request context")
}

// rateFromContext returns a rate value read from the context, or our default rate
func rateFromContext(ctx context.Context) int {
	if rate, ok := ctx.Value(rateKey).(int); ok {
		return rate
	}

	return defaultRate
}
