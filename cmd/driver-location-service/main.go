package driver_location_service

import (
	"context"
	"log"
	"net/http"
	"ride-hail/internal/common/config"
	commonrmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/handler"
	"ride-hail/internal/driver/repository"
	driverrmq "ride-hail/internal/driver/rmq"
	"ride-hail/internal/driver/service"
	ws "ride-hail/internal/driver/websocket"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func DriverMain(cfg *config.Config, conn *pgx.Conn, commonMq *commonrmq.RabbitMQ, mux *http.ServeMux) {
	log.Println("Starting Driver & Location Service...")

	rmqClient, err := driverrmq.NewClient(commonMq.URL, "driver_topic")
	if err != nil {
		log.Fatalf("failed to init driver rmq client: %v", err)
	}
	defer rmqClient.Close()

	repo := repository.NewDriverRepository(conn)

	hub := websocket.NewHub()
	go hub.Run()

	svc := service.NewDriverService(repo, rmqClient, hub)

	go svc.ListenForRides(context.Background(), "ride_topic")

	h := handler.NewHandler(svc)
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.GoOnline)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.GoOffline)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.Location)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.Start)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.Complete)

	mux.HandleFunc("/ws/drivers/", func(w http.ResponseWriter, r *http.Request) {
		ws.DriverWSHandler(w, r, hub)
	})

	addr := ":" + strconv.Itoa(cfg.Services.DriverLocationServicePort)
	log.Printf("Driver service running on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("driver-status-service failed: %v", err)
	}
}
