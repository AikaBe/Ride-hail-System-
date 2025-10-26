package driver_location_service

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"ride-hail/internal/common/config"
	commonrmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/handler"
	"ride-hail/internal/driver/repository"
	driverrmq "ride-hail/internal/driver/rmq"
	"ride-hail/internal/driver/service"
)

func RunDriver(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ, mux *http.ServeMux, hub *websocket.Hub) {
	log.Println("Starting Driver & Location Service...")

	rmqClient, err := driverrmq.NewClient(commonMq.URL, "driver_topic")
	if err != nil {
		log.Fatalf("failed to init driver rmq client: %v", err)
	}

	repo := repository.NewDriverRepository(conn)

	svc := service.NewDriverService(repo, rmqClient, hub)

	go svc.ListenForRides(context.Background(), "ride_topic")

	h := handler.NewHandler(svc)
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.GoOnline)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.GoOffline)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.Location)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.Start)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.Complete)
}
