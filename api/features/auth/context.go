package auth

import "context"

// principalCtxKey is an unexported context key type so principal values in the
// request context cannot collide with keys set by other packages.
type principalCtxKey struct{}

// WithPrincipal returns a copy of ctx carrying the authenticated principal.
// It is set by RequireAuth once a session has been verified.
func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal stored in ctx.
// The bool is false when no principal is present (i.e. the request did not pass
// through RequireAuth). Handlers behind RequireAuth can rely on ok being true.
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalCtxKey{}).(*Principal)
	return principal, ok
}
