package ride_service

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/mq"
)

func Run(cfg *config.Config, pool *pgxpool.Pool, mq *mq.RabbitMQ) {
	log.Printf("✅ Ride Service running on port %d\n", cfg.Services.RideServicePort)
}
