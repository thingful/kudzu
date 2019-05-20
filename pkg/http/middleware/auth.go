package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/thingful/kudzu/pkg/http/handlers"
	"github.com/thingful/kudzu/pkg/postgres"
)

const (
	// BearerSchema is the prefix for bearer auth
	BearerSchema = "Bearer "

	subjectKey = contextKey("subject")
	rolesKey   = contextKey("roles")
)

// AppLoader is an interface we define here for a type that can load an App from
// somewhere to verify the validity of the submitted token.
type AppLoader interface {
	LoadApp(context.Context, string) (*postgres.App, error)
}

// AuthMiddleware is middleware that checks in the request for an authorization
// header which we attempt to validate against the DB
type AuthMiddleware struct {
	al AppLoader
}

// NewAuthMiddleware returns a new AuthMiddleware instance. Takes as input an
// AppLoader which is a type that exposes a single method for loading an App on
// being given a token.
func NewAuthMiddleware(al AppLoader) *AuthMiddleware {
	return &AuthMiddleware{
		al: al,
	}
}

// Handler is the middleware handler function
func (a *AuthMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token, err := extractToken(r)
		if err != nil {
			invalidTokenError(w, err)
			return
		}

		app, err := a.al.LoadApp(ctx, token)
		if err != nil {
			authForbiddenError(w, err)
			return
		}

		ctx = context.WithValue(ctx, subjectKey, app.UID)
		ctx = context.WithValue(ctx, rolesKey, app.Roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

// extractToken attempts to extract the token from the Authorization header and
// returns either that or an error
func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header is required")
	}

	if !strings.HasPrefix(authHeader, BearerSchema) {
		return "", errors.New("Authorization requires Bearer scheme")
	}

	return authHeader[len(BearerSchema):], nil
}

func invalidTokenError(w http.ResponseWriter, err error) {
	httpErr := &handlers.HTTPError{
		Code: http.StatusUnauthorized,
		Err:  err,
	}

	b, err := json.Marshal(httpErr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(b)
}

func authForbiddenError(w http.ResponseWriter, err error) {
	httpErr := &handlers.HTTPError{
		Code: http.StatusForbidden,
		Err:  err,
	}

	b, err := json.Marshal(httpErr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write(b)
}

// RolesFromContext returns a slice of strings from the context representing the
// roles associated with the current user. Returns an empty slice if no roles
// are found.
func RolesFromContext(ctx context.Context) postgres.ScopeClaims {
	if roles, ok := ctx.Value(rolesKey).(postgres.ScopeClaims); ok {
		return roles
	}

	return postgres.ScopeClaims{}
}
