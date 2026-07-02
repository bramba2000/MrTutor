package auth

import (
	"context"
	"net/http"
)

// HasRole checks if the user has the specified role. Admin have all roles.
func HasRole(ctx context.Context, role UserType) bool {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return false
	}
	return principal.Role == role || principal.Role == AdminUserType
}

// RequireRole is a middleware that checks if the user has the specified role. Admin have all roles.
func RequireRole(role string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasRole(r.Context(), UserType(role)) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
