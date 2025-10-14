package http

import (
	"encoding/json"
	"net/http"
	"ride-hail/internal/ride/service"
)

type RideHandler struct {
	RideManager *service.RideManager
}

func NewRideHandler(manager *service.RideManager) *RideHandler {
	return &RideHandler{RideManager: manager}
}

func (h *RideHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
	var req service.RideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.RideManager.CreateRide(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *RideHandler) CancelRide(w http.ResponseWriter, r *http.Request) {
	rideID := r.URL.Query().Get("ride_id")
	if rideID == "" {
		http.Error(w, "ride_id is required", http.StatusBadRequest)
		return
	}

	var req service.CancelRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.RideManager.CancelRide(r.Context(), rideID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}