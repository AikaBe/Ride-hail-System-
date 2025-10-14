package ride_service

import (
	"log"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/mq"

	"github.com/jackc/pgx/v5"
)

func Run(cfg *config.Config, conn *pgx.Conn, mq *mq.RabbitMQ) {
	log.Printf("✅ Ride Service running on port %d\n", cfg.Services.RideServicePort)
}
