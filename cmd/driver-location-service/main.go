package driver_location_service

import (
	"net/http"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/driver/handler"
	"ride-hail/internal/driver/repository"
	"ride-hail/internal/driver/service"

	"github.com/jackc/pgx/v5"
)

func DriverMain(cfg *config.Config, conn *pgx.Conn) {
	repo := repository.NewDriverRepository(conn)
	svc := service.NewDriverService(repo)
	h := handler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.GoOnline)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.GoOffline)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.Location)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.Start)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.Complete)

	serverAddr := ":8082"
	logger.Info("service_started", "Driver Status Service running", "init-request", "")

	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		logger.Error("service_failed", "Driver Status Service failed to start", "init-request", "", err.Error(), "")
	}
}
