package auth

import (
	"mrtutor-api/transport/httpbind"
	"net/http"
)

type controller struct {
	service Service
}

func (c controller) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/auth/login", httpbind.NewHandler(
		httpbind.NewJSONDecoder[LoginRequest](),
		c.service.Login,
		func(w http.ResponseWriter, sessionToken string) error {
			http.SetCookie(w, NewSessionToken(Session{Token: sessionToken}))
			return nil
		},
	))
	mux.Handle("/auth/logout", httpbind.NewNoOutputHandler(
		func(r *http.Request) (string, error) {
			cookie, err := r.Cookie("session")
			if err != nil {
				return "", err
			}
			return cookie.Value, nil
		},
		c.service.Logout,
		func(w http.ResponseWriter) error {
			w.WriteHeader(http.StatusOK)
			http.SetCookie(w, &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			})
			return nil
		},
	))
	mux.Handle("/auth/register", httpbind.NewHandler(
		httpbind.NewJSONDecoder[RegisterRequest](),
		c.service.Register,
		func(w http.ResponseWriter, out *RegisterResponse) error {
			http.SetCookie(w, NewSessionToken(Session{Token: out.SessionToken}))
			return httpbind.NewJSONEncoder[Principal](http.StatusCreated)(w, out.Principal)
		},
	))
}

func NewSessionToken(session Session) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		HttpOnly: true,
	}
}
