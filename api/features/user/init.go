package user

import (
	"database/sql"
	"log/slog"
	"net/http"
)

type Module struct {
	SessionManager SessionManager
	Service        UserService
	controller     UserController
}

func (m Module) RegisterRoutes(mux *http.ServeMux) {
	m.controller.RegisterRoutes(mux)
}

func NewModule(
	db *sql.DB,
	logger *slog.Logger,
) Module {
	sessionStore := newSessionStore(db)
	sessionManager := NewSessionManager(sessionStore)

	userRepo := newSQLUserRepository(db)
	userService := NewUserService(userRepo, sessionManager, logger)
	userController := NewUserController(userService)

	return Module{
		SessionManager: sessionManager,
		Service:        userService,
		controller:     userController,
	}
}
