package ride_service

import (
	"net/http"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/rmq"
	ridehttp "ride-hail/internal/ride/handler"
	"ride-hail/internal/ride/repository"
	"ride-hail/internal/ride/service"

	"github.com/jackc/pgx/v5"
)

func Run(cfg *config.Config, conn *pgx.Conn, mq *rmq.RabbitMQ) {
	logger.SetServiceName("ride-service")

	repo := repository.NewRideRepository(conn)
	service := service.NewRideManager(repo)
	handler := ridehttp.NewRideHandler(service)

	mux := http.NewServeMux()
	ridehttp.SetupRoutes(mux, handler)

	logger.Info("server_listening", "HTTP server listening on port 8080", "init-request", "")
	http.ListenAndServe(":8080", mux)

	   if err := http.ListenAndServe(":8080", mux); err != nil {
        logger.Error("server_failed", "Ride Service failed to start", "init-request", "", err.Error(), "")
        return
    }
}
