package admin_service

import (
	"net/http"

	"ride-hail/internal/admin/handler"
	"ride-hail/internal/admin/repository"
	"ride-hail/internal/admin/service"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/logger"

	"github.com/jackc/pgx/v5"
)

func RunAdmin(cfg *config.Config, conn *pgx.Conn, mux *http.ServeMux) {
	logger.SetServiceName("admin-service")

	logger.Info("startup", "Starting Admin Service...", "", "")

	repo := repository.NewAdminRepository(conn)
	svc := service.NewAdminService(repo)
	h := handler.NewAdminHandler(svc)

	mux.HandleFunc("GET /admin/overview", h.GetSystemOverview)
	mux.HandleFunc("GET /admin/rides/active", h.GetActiveRides)
	mux.HandleFunc("GET /admin/drivers/online", h.GetOnlineDrivers)
	mux.HandleFunc("GET /admin/metrics", h.GetSystemMetrics)

	logger.Info("startup_complete", "Admin Service started successfully", "", "")
}
