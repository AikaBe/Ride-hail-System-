package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	cmdDriver "ride-hail/cmd/driver-location-service"
	cmdRide "ride-hail/cmd/ride-service"
	cmdUser "ride-hail/cmd/user-service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	driverws "ride-hail/internal/driver/websocket"
	ridews "ride-hail/internal/ride/websocket"
	"ride-hail/internal/user/jwt"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	pg, err := db.NewPostgres(
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer pg.Close()

	if err := pg.RunMigrations("migrations"); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	commonRMQ, err := rmq.NewRabbitMQ(
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
	)
	if err != nil {
		log.Fatalf("rabbitmq error: %v", err)
	}
	defer commonRMQ.Close()

	jwtManager := jwt.NewManager("super-secret-key", 15*time.Minute, 7*24*time.Hour)

	hub := websocket.NewHub()
	go hub.Run()

	mux := http.NewServeMux()

	cmdRide.RunRide(cfg, pg.Conn, commonRMQ, mux, hub)
	cmdDriver.RunDriver(cfg, pg.Conn, commonRMQ, mux, hub)
	cmdUser.RunUser(pg.Conn, mux, jwtManager)

	wsMux := http.NewServeMux()

	wsMux.HandleFunc("/ws/passengers/", func(w http.ResponseWriter, r *http.Request) {
		ridews.PassengerWSHandler(w, r, hub, jwtManager)
	})
	wsMux.HandleFunc("/ws/drivers/", func(w http.ResponseWriter, r *http.Request) {
		driverws.DriverWSHandler(w, r, hub, jwtManager)
	})

	go func() {
		log.Println("WebSocket server running on ws://localhost:3000")
		if err := http.ListenAndServe(":3000", wsMux); err != nil {
			log.Fatalf("WebSocket server error: %v", err)
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
		log.Println("🚀 All services are up on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-stop
	log.Println("⏹ Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	log.Println("✅ Shutdown complete")
}
