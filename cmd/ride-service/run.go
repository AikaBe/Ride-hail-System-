package ride_service

import (
	"context"
	"net/http"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/logger"
	commonrmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	ridehttp "ride-hail/internal/ride/handler"
	"ride-hail/internal/ride/repository"
	ridermq "ride-hail/internal/ride/rmq"
	"ride-hail/internal/ride/service"
	ridews "ride-hail/internal/ride/websocket"
	"ride-hail/internal/user/jwt"

	"github.com/jackc/pgx/v5"
)

func RunRide(
	cfg *config.Config,
	conn *pgx.Conn,
	commonMq *commonrmq.RabbitMQ,
	mux *http.ServeMux,
	hub *websocket.Hub,
	wsMux *http.ServeMux,
	jwtManager *jwt.Manager,
) {
	logger.SetServiceName("ride-service")

	logger.Info("startup", "Starting Ride Service...", "", "")

	rmqClient, err := ridermq.NewClient(commonMq.URL, "ride_topic")
	if err != nil {
		logger.Error("init_rmq_client", "Failed to init ride RMQ client", "", "", err.Error())
		return
	}

	repo := repository.NewRideRepository(conn)
	svc := service.NewRideManager(repo, rmqClient, hub)
	h := ridehttp.NewRideHandler(svc)

	go func() {
		logger.Info("listener_driver", "Listening for driver responses...", "", "")
		svc.ListenForDriver(context.Background(), "driver_responses")
	}()

	go func() {
		logger.Info("listener_location", "Listening for location updates...", "", "")
		svc.LocationUpdate(context.Background(), "location_updates_ride")
	}()

	mux.HandleFunc("POST /rides", h.CreateRide)
	mux.HandleFunc("POST /rides/{ride_id}/cancel", h.CancelRide)

	wsMux.HandleFunc("/ws/passengers/", func(w http.ResponseWriter, r *http.Request) {
		ridews.PassengerWSHandler(w, r, hub, jwtManager, svc)
	})

	logger.Info("startup_complete", "Ride Service started successfully", "", "")
}
