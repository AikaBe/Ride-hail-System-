package user_service

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"ride-hail/internal/user/handler"
	token "ride-hail/internal/user/jwt"
	"ride-hail/internal/user/repository"
	"ride-hail/internal/user/service"
	"time"
)

func Run(db *pgx.Conn, mux *http.ServeMux) {
	userRepo := repository.NewUserRepository(db)
	tokenManager := token.NewManager("supersecret", 15*time.Minute, 7*24*time.Hour)
	authService := service.NewAuthService(userRepo, tokenManager)
	authHandler := handler.NewAuthHandler(authService)

	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)
}
