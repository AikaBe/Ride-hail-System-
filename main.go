package main

import (
	"log"
	"net/http"
	cmdDriver "ride-hail/cmd/driver-location-service"
	cmdRide "ride-hail/cmd/ride-service"
	cmdUser "ride-hail/cmd/user-service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/rmq"
	"ride-hail/internal/user/handler"
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

	rmq, err := rmq.NewRabbitMQ(
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
	)
	if err != nil {
		log.Fatalf("rabbitmq error: %v", err)
	}
	defer rmq.Close()

	go func() {
		log.Println("WebSocket server running on ws://localhost:3001")
		if err := http.ListenAndServe(":3001", nil); err != nil {
			log.Fatalf("WebSocket server error: %v", err)
		}
	}()
	go func() {
		log.Println("HTTP server running on :8085")
		if err := http.ListenAndServe(":8085", nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
	mux := http.NewServeMux()
	go cmdUser.Run(pg.Conn, mux)
	go cmdRide.Run(cfg, pg.Conn, rmq, mux)
	go cmdDriver.DriverMain(cfg, pg.Conn, rmq, mux)
	select {}
}
