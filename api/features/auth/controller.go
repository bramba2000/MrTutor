package auth

import (
	"mrtutor/api/config"
	"mrtutor/api/transport/httpbind"
	"net/http"
)

const sessionCookieName = "session"

type controller struct {
	service Service
}

func (c controller) LoginHandler() http.Handler {
	return httpbind.NewHandler(
		httpbind.NewJSONDecoder[LoginRequest](),
		c.service.Login,
		func(w http.ResponseWriter, sessionToken string) error {
			http.SetCookie(w, NewSessionToken(Session{Token: sessionToken}))
			return nil
		},
	)
}

func (c controller) LogoutHandler() http.Handler {
	return httpbind.NewNoOutputHandler(
		func(r *http.Request) (string, error) {
			cookie, err := r.Cookie("session")
			if err != nil {
				if err == http.ErrNoCookie {
					return "", nil
				}
				return "", err
			}
			return cookie.Value, nil
		},
		c.service.Logout,
		func(w http.ResponseWriter) error {
			w.WriteHeader(http.StatusOK)
			http.SetCookie(w, &http.Cookie{
				Name:   sessionCookieName,
				Value:  "",
				MaxAge: -1,
			})
			return nil
		},
	)
}

func (c controller) RegisterHandler() http.Handler {
	return httpbind.NewHandler(
		httpbind.NewJSONDecoder[RegisterRequest](),
		c.service.Register,
		func(w http.ResponseWriter, out *RegisterResponse) error {
			http.SetCookie(w, NewSessionToken(Session{Token: out.SessionToken}))
			return httpbind.NewJSONEncoder[Principal](http.StatusCreated)(w, out.Principal)
		},
	)
}

// RequireAuth authenticates the request from its session cookie and, on success,
// stores the resolved principal in the request context (retrieve it with
// PrincipalFromContext). It responds 401 and stops the chain when the session is
// missing or invalid, so downstream handlers can assume an authenticated principal.
//
// It is the authentication layer of the security model; fine-grained
// authorization (roles, ownership) is a separate service-level concern.
func (c controller) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			token = cookie.Value
		}

		principal, err := c.service.VerifySession(r.Context(), token)
		if err != nil || principal == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c controller) MeHandler() http.Handler {
	type PrincipalResponse struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	// Behind RequireAuth, so the principal is always present in the context.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		response := PrincipalResponse{
			ID:       principal.ID,
			Username: principal.Username,
			Email:    principal.Email,
		}
		if err := httpbind.NewJSONEncoder[PrincipalResponse](http.StatusOK)(w, response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (c controller) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /auth/login", c.LoginHandler())
	mux.Handle("POST /auth/logout", c.LogoutHandler())
	mux.Handle("POST /auth/register", c.RegisterHandler())
	mux.Handle("GET /auth/me", c.RequireAuth(c.MeHandler()))
}

func NewSessionToken(session Session) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		HttpOnly: true,
		Secure:   config.Mode == config.PROD,
		SameSite: http.SameSiteLaxMode,
	}
}
