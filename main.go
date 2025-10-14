package main

import (
	"log"
	cmdDriver "ride-hail/cmd/driver-location-service"
	cmdRide "ride-hail/cmd/ride-service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/mq"
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

	cmdRide.Run(cfg, pg.Conn, rmq)
	cmdDriver.DriverMain(cfg, pg.Conn)
}
