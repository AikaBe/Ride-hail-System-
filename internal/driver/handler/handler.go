package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/model"
)

type DriverService interface {
	GoOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error)
	GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error)
	Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error)
	Start(ctx context.Context, driverID string, rideId string, location model.Location) (model.StartResponse, error)
	Complete(ctx context.Context, driverID string, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error)
}

type Handler struct {
	service DriverService
}

func NewHandler(s DriverService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) GoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	requestID := r.Header.Get("X-Request-ID")

	logger.Info("driver_request_received", "GoOnline request received", requestID, driverID)

	var req model.OnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid_request_body", "Failed to decode OnlineRequest", requestID, driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GoOnline(ctx, driverID, req.Latitude, req.Longitude)
	if err != nil {
		logger.Error("service_call_failed", "GoOnline service error", requestID, driverID, err.Error(), "")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("driver_request_success", "GoOnline completed successfully", requestID, driverID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	requestID := r.Header.Get("X-Request-ID")

	logger.Info("driver_request_received", "GoOffline request received", requestID, driverID)

	resp, err := h.service.GoOffline(ctx, driverID)
	if err != nil {
		logger.Error("service_call_failed", "GoOffline service error", requestID, driverID, err.Error(), "")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("driver_request_success", "GoOffline completed successfully", requestID, driverID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Location(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	requestID := r.Header.Get("X-Request-ID")

	logger.Info("driver_request_received", "Location request received", requestID, driverID)

	var req model.LocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid_request_body", "Failed to decode LocationRequest", requestID, driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Location(ctx, driverID, req)
	if err != nil {
		logger.Error("service_call_failed", "Location service error", requestID, driverID, err.Error(), "")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("driver_request_success", "Location completed successfully", requestID, driverID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	requestID := r.Header.Get("X-Request-ID")

	logger.Info("driver_request_received", "Start request received", requestID, driverID)

	var req model.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid_request_body", "Failed to decode StartRequest", requestID, driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.DriverLocation.Latitude,
		Longitude: req.DriverLocation.Longitude,
	}

	resp, err := h.service.Start(ctx, driverID, req.RideID, location)
	if err != nil {
		logger.Error("service_call_failed", "Start service error", requestID, driverID, err.Error(), "")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("driver_request_success", "Start completed successfully", requestID, driverID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	requestID := r.Header.Get("X-Request-ID")

	logger.Info("driver_request_received", "Complete request received", requestID, driverID)

	var req model.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid_request_body", "Failed to decode CompleteRequest", requestID, driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.FinalLocation.Latitude,
		Longitude: req.FinalLocation.Longitude,
	}

	resp, err := h.service.Complete(ctx, driverID, req, location)
	if err != nil {
		logger.Error("service_call_failed", "Complete service error", requestID, driverID, err.Error(), "")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("driver_request_success", "Complete completed successfully", requestID, driverID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
