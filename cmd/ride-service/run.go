package ride_service

import (
	"log"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/mq"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(cfg *config.Config, pool *pgxpool.Pool, mq *mq.RabbitMQ) {
	log.Printf("✅ Ride Service running on port %d\n", cfg.Services.RideServicePort)
}
