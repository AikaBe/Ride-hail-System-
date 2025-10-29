package user_service

import (
	"net/http"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/user/handler"
	"ride-hail/internal/user/jwt"
	"ride-hail/internal/user/repository"
	"ride-hail/internal/user/service"

	"github.com/jackc/pgx/v5"
)

func RunUser(db *pgx.Conn, mux *http.ServeMux, jwtManager *jwt.Manager) {
	logger.SetServiceName("user-service")

	logger.Info("startup", "Starting User Service...", "", "")

	userRepo := repository.NewUserRepository(db)
	if userRepo == nil {
		logger.Error("init_repository", "Failed to initialize user repository", "", "", "repository is nil")
		return
	}

	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /refresh", authHandler.RefreshToken)

	logger.Info("startup_complete", "User Service started successfully", "", "")
}
