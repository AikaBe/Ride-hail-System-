package driver_location_service

import (
	"context"
	"net/http"

	"ride-hail-system/internal/common/config"
	"ride-hail-system/internal/common/logger"
	commonrmq "ride-hail-system/internal/common/rmq"
	"ride-hail-system/internal/common/websocket"
	"ride-hail-system/internal/driver/handler"
	"ride-hail-system/internal/driver/repository"
	driverrmq "ride-hail-system/internal/driver/rmq"
	"ride-hail-system/internal/driver/service"
	driverws "ride-hail-system/internal/driver/websocket"
	"ride-hail-system/internal/user/jwt"

	"github.com/jackc/pgx/v5"
)

func RunDriver(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ, mux *http.ServeMux, hub *websocket.Hub, wsMux *http.ServeMux, jwtManager *jwt.Manager) {
	logger.SetServiceName("driver-location-service")

	logger.Info("startup", "Starting Driver & Location Service...", "", "")

	rmqClient, err := driverrmq.NewClient(commonMq.URL, "driver_topic")
	if err != nil {
		logger.Error("init_rmq_client", "Failed to init driver RMQ client", "", "", err.Error())
		return
	}

	repo := repository.NewDriverRepository(conn)
	svc := service.NewDriverService(repo, rmqClient, hub)
	h := handler.NewHandler(svc, jwtManager)

	mux.HandleFunc("POST /drivers/{driver_id}/online", h.GoOnline)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.GoOffline)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.UpdateLocation)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.Start)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.Complete)

	wsMux.HandleFunc("/ws/drivers/", func(w http.ResponseWriter, r *http.Request) {
		driverws.DriverWSHandler(w, r, hub, jwtManager, svc)
	})

	go func() {
		logger.Info("listener_rides", "Listening for ride requests...", "", "")
		svc.ListenForRides(context.Background(), "ride_requests")
	}()

	go func() {
		logger.Info("listener_passengers", "Listening for passenger matching...", "", "")
		svc.ListenForPassengers(context.Background(), "driver_matching")
	}()

	logger.Info("startup_complete", "Driver & Location Service started successfully", "", "")
}
