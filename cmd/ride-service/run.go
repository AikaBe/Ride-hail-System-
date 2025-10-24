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

func Run(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ) {
	log.Printf("Ride Service running on port %d\n", cfg.Services.RideServicePort)

	rmqClient, err := ridermq.NewClient(commonMq.URL, "ride_topic")
	if err != nil {
		log.Fatalf("failed to init driver rmq client: %v", err)
	}
	defer rmqClient.Close()

	repo := repository.NewRideRepository(conn)

	hub := websocket.NewHub()
	go hub.Run()

	service := service.NewRideManager(repo, rmqClient, hub)

	go service.ListenForRides(context.Background(), "driver_responses")

	handler := ridehttp.NewRideHandler(service)

	mux := http.NewServeMux()
	ridehttp.SetupRoutes(mux, handler)

	mux.HandleFunc("/ws/drivers/", func(w http.ResponseWriter, r *http.Request) {
		ws.PassengerWSHandler(w, r, hub)
	})

	log.Println("🚀 Server running on port 8080")
	http.ListenAndServe(":8080", mux)
}
