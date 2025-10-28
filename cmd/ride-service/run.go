package ride_service

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"ride-hail/internal/common/config"
	commonrmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	ridehttp "ride-hail/internal/ride/handler"
	"ride-hail/internal/ride/repository"
	ridermq "ride-hail/internal/ride/rmq"
	"ride-hail/internal/ride/service"
	ridews "ride-hail/internal/ride/websocket"
	"ride-hail/internal/user/jwt"
)

func RunRide(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ, mux *http.ServeMux, hub *websocket.Hub, wsMux *http.ServeMux, jwtManager *jwt.Manager) {
	log.Printf("Ride Service running on port %d\n", cfg.Services.RideServicePort)

	rmqClient, err := ridermq.NewClient(commonMq.URL, "ride_topic")
	if err != nil {
		log.Fatalf("failed to init driver rmq client: %v", err)
	}

	repo := repository.NewRideRepository(conn)
	service := service.NewRideManager(repo, rmqClient, hub)
	handler := ridehttp.NewRideHandler(service)

	go service.ListenForDriver(context.Background(), "driver_responses")
	go service.LocationUpdate(context.Background(), "location_updates_ride")

	mux.HandleFunc("POST /rides", handler.CreateRide)
	mux.HandleFunc("POST /rides/{ride_id}/cancel", handler.CancelRide)

	wsMux.HandleFunc("/ws/passengers/", func(w http.ResponseWriter, r *http.Request) {
		ridews.PassengerWSHandler(w, r, hub, jwtManager, service)
	})
}
