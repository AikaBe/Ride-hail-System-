package ride_service

import (
	"context"
	"log"
	"net/http"
	"ride-hail/internal/common/config"
	commonrmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	ridehttp "ride-hail/internal/ride/handler"
	"ride-hail/internal/ride/repository"
	ridermq "ride-hail/internal/ride/rmq"
	"ride-hail/internal/ride/service"
	ws "ride-hail/internal/ride/websocket"

	"github.com/jackc/pgx/v5"
)

func RunRide(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ, mux *http.ServeMux) {
	log.Printf("Ride Service running on port %d\n", cfg.Services.RideServicePort)

	rmqClient, err := ridermq.NewClient(commonMq.URL, "ride_topic")
	if err != nil {
		log.Fatalf("failed to init driver rmq client: %v", err)
	}
	defer rmqClient.Close()

	hub := websocket.NewHub()
	go hub.Run()

	repo := repository.NewRideRepository(conn)
	service := service.NewRideManager(repo, rmqClient, hub)
	handler := ridehttp.NewRideHandler(service)

	go service.ListenForRides(context.Background(), "driver_responses")

	mux.HandleFunc("POST /rides", handler.CreateRide)
	mux.HandleFunc("POST /rides/{ride_id}/cancel", handler.CancelRide)

	mux.HandleFunc("/ws/drivers/", func(w http.ResponseWriter, r *http.Request) {
		ws.PassengerWSHandler(w, r, hub)
	})
}
