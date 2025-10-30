package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"ride-hail/internal/admin/model"
	"ride-hail/internal/admin/service"

	"ride-hail/internal/common/logger"
)

type AdminHandler struct {
	service *service.AdminService
}

func NewAdminHandler(service *service.AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) GetSystemOverview(w http.ResponseWriter, r *http.Request) {
	const action = "GetSystemOverview"
	requestID := r.Header.Get("X-Request-ID")

	overview, err := h.service.GetSystemOverview(r.Context())
	if err != nil {
		logger.Error(action, "Failed to get system overview", requestID, "", err.Error(), "")
		http.Error(w, "Failed to get system overview", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(overview); err != nil {
		logger.Error(action, "Failed to encode response", requestID, "", err.Error(), "")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logger.Info(action, "System overview retrieved successfully", requestID, "")
}

func (h *AdminHandler) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	const action = "GetActiveRides"
	requestID := r.Header.Get("X-Request-ID")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}

	response, err := h.service.GetActiveRides(r.Context(), page, pageSize)
	if err != nil {
		logger.Error(action, "Failed to get active rides", requestID, "", err.Error(), "")
		http.Error(w, "Failed to get active rides", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error(action, "Failed to encode response", requestID, "", err.Error(), "")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logger.Info(action, "Active rides retrieved successfully", requestID, "")
}

func (h *AdminHandler) GetOnlineDrivers(w http.ResponseWriter, r *http.Request) {
	const action = "GetOnlineDrivers"
	requestID := r.Header.Get("X-Request-ID")

	drivers, err := h.service.GetOnlineDrivers(r.Context())
	if err != nil {
		logger.Error(action, "Failed to get online drivers", requestID, "", err.Error(), "")
		http.Error(w, "Failed to get online drivers", http.StatusInternalServerError)
		return
	}

	if drivers == nil {
		drivers = make([]model.OnlineDriver, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(drivers); err != nil {
		logger.Error(action, "Failed to encode response", requestID, "", err.Error(), "")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logger.Info(action, "Online drivers retrieved successfully", requestID, "")
}

func (h *AdminHandler) GetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	const action = "GetSystemMetrics"
	requestID := r.Header.Get("X-Request-ID")

	metrics, err := h.service.GetSystemMetrics(r.Context())
	if err != nil {
		metrics = &model.SystemMetrics{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		logger.Error(action, "Failed to encode response", requestID, "", err.Error(), "")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logger.Info(action, "System metrics retrieved successfully", requestID, "")
}
