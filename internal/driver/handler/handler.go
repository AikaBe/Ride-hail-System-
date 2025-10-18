package handler

import (
	"context"
	"encoding/json"
	"net/http"
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

	var req model.OnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GoOnline(ctx, driverID, req.Latitude, req.Longitude)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	resp, err := h.service.GoOffline(ctx, driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Location(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req model.LocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Location(ctx, driverID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req model.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.DriverLocation.Latitude,
		Longitude: req.DriverLocation.Longitude,
	}

	resp, err := h.service.Start(ctx, driverID, req.RideID, location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req model.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.FinalLocation.Latitude,
		Longitude: req.FinalLocation.Longitude,
	}

	resp, err := h.service.Complete(ctx, driverID, req, location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
