package admin_service

import (
	"log"
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
	log.Println("Starting Admin Service...")

	repo := repository.NewAdminRepository(conn)
	svc := service.NewAdminService(repo)
	h := handler.NewAdminHandler(svc)

	// Admin API routes
	mux.HandleFunc("GET /admin/overview", h.GetSystemOverview)
	mux.HandleFunc("GET /admin/rides/active", h.GetActiveRides)
	mux.HandleFunc("GET /admin/drivers/online", h.GetOnlineDrivers)
	mux.HandleFunc("GET /admin/metrics", h.GetSystemMetrics)

	log.Printf("Admin Service running and registered routes")
}
