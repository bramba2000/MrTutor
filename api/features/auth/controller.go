package auth

import (
	"mrtutor-api/transport/httpbind"
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

func (c controller) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/auth/login", c.LoginHandler())
	mux.Handle("/auth/logout", c.LogoutHandler())
	mux.Handle("/auth/register", c.RegisterHandler())
}

func NewSessionToken(session Session) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		HttpOnly: true,
	}
}
