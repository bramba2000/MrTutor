package user

import (
	"mrtutor-api/transport/httpbind"
	"net/http"
)

type UserController struct {
	service UserService
}

func (c UserController) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /users", httpbind.Handler(
		httpbind.NewJSONDecoder[CreateUserParams](),
		c.service.CreateUser,
		httpbind.NewJSONEncoder[User](http.StatusCreated),
	))
	mux.Handle("POST /auth/login", httpbind.Handler(
		httpbind.NewJSONDecoder[LoginUserParam](),
		c.service.LoginUser,
		func(w http.ResponseWriter, session string) error {
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    session,
				HttpOnly: true,
			})
			w.WriteHeader(http.StatusOK)
			return nil
		},
	))
}

func NewUserController(service UserService) UserController {
	return UserController{
		service,
	}
}
