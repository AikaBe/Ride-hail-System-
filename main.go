package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmdAdmin "ride-hail/cmd/admin-service"
	cmdDriver "ride-hail/cmd/driver-location-service"
	cmdRide "ride-hail/cmd/ride-service"
	cmdUser "ride-hail/cmd/user-service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/user/jwt"
)

func main() {
	logger.SetServiceName("main-service")

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("init_config", "failed to load configuration", "", "", err.Error())
		os.Exit(1)
	}
	logger.Info("init_config", "configuration successfully loaded", "", "")

	pg, err := db.NewPostgres(
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
	if err != nil {
		logger.Error("init_db", "failed to connect to PostgreSQL", "", "", err.Error())
		os.Exit(1)
	}
	defer pg.Close()
	logger.Info("init_db", "PostgreSQL connected successfully", "", "")

	if err := pg.RunMigrations("migrations"); err != nil {
		logger.Error("migrations", "failed to run migrations", "", "", err.Error())
		os.Exit(1)
	}
	logger.Info("migrations", "database migrations completed", "", "")

	commonRMQ, err := rmq.NewRabbitMQ(
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
	)
	if err != nil {
		logger.Error("init_rabbitmq", "failed to connect to RabbitMQ", "", "", err.Error())
		os.Exit(1)
	}
	defer commonRMQ.Close()
	logger.Info("init_rabbitmq", "RabbitMQ connection established", "", "")

	jwtManager := jwt.NewManager("super-secret-key", 15*time.Minute, 7*24*time.Hour)
	logger.Info("init_jwt", "JWT manager initialized", "", "")

	hub := websocket.NewHub()
	go hub.Run()
	logger.Info("init_websocket", "WebSocket hub started", "", "")

	mux := http.NewServeMux()
	wsMux := http.NewServeMux()

	go cmdUser.RunUser(pg.Conn, mux, jwtManager)
	go cmdRide.RunRide(cfg, pg.Conn, commonRMQ, mux, hub, wsMux, jwtManager)
	go cmdDriver.RunDriver(cfg, pg.Conn, commonRMQ, mux, hub, wsMux, jwtManager)
	go cmdAdmin.RunAdmin(cfg, pg.Conn, mux)
	logger.Info("run_services", "all microservices initialized", "", "")

	go func() {
		logger.Info("websocket_server", "WebSocket server running on ws://localhost:3000", "", "")
		if err := http.ListenAndServe(":3000", wsMux); err != nil && err != http.ErrServerClosed {
			logger.Error("websocket_server", "server failed", "", "", err.Error())
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("http_server", "All services are up on port 8080", "", "")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http_server", "server failed", "", "", err.Error())
			os.Exit(1)
		}
	}()

	<-stop
	logger.Warn("shutdown", "received stop signal, shutting down gracefully...", "", "", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "HTTP server forced to shutdown", "", "", err.Error())
	} else {
		logger.Info("shutdown", "all services stopped successfully", "", "")
	}
}
