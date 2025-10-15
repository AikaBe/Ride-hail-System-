package main

import (
	"log"
	"net/http"
	cmdDriver "ride-hail/cmd/driver-location-service"
	cmdRide "ride-hail/cmd/ride-service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/mq"
	"ride-hail/internal/common/registration"
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

	rmq, err := mq.NewRabbitMQ(
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password,
	)
	if err != nil {
		log.Fatalf("rabbitmq error: %v", err)
	}
	defer rmq.Close()

	http.HandleFunc("/register", registration.RegisterHandler(pg.Conn))

	go func() {
		log.Println("HTTP server running on :8085")
		if err := http.ListenAndServe(":8085", nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	cmdRide.Run(cfg, pg.Conn, rmq)
	cmdDriver.DriverMain(cfg, pg.Conn)
}
