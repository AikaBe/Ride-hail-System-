package ride_service

import (
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/rmq"
	ridehttp "ride-hail/internal/ride/handler"
	"ride-hail/internal/ride/repository"
	"ride-hail/internal/ride/service"
)

func Run(cfg *config.Config, conn *pgx.Conn, mq *rmq.RabbitMQ) {
	log.Printf("✅ Ride Service running on port %d\n", cfg.Services.RideServicePort)

	repo := repository.NewRideRepository(conn)
	service := service.NewRideManager(repo)
	handler := ridehttp.NewRideHandler(service)

	mux := http.NewServeMux()
	ridehttp.SetupRoutes(mux, handler)

	log.Println("🚀 Server running on port 8080")
	http.ListenAndServe(":8080", mux)
}
