package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ride-hail/internal/ride/service"
)

type RideHandler struct {
	RideService *service.RideService
}

func NewRideHandler(service *service.RideService) *RideHandler {
	return &RideHandler{RideService: service}
}

func (h *RideHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
	var req RideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	ride, pickup, destination, err := MapRideRequestToEntities(req)
	if err != nil {
		http.Error(w, "invalid request mapping", http.StatusBadRequest)
		return
	}

	createdRide, distance, duration, err := h.RideService.CreateRide(r.Context(), ride, pickup, destination)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create ride: %v", err), http.StatusInternalServerError)
		return
	}

	resp := RideResponse{
		RideID:                   string(createdRide.ID),
		RideNumber:               createdRide.RideNumber,
		Status:                   string(*createdRide.Status),
		EstimatedFare:            *createdRide.EstimatedFare,
		EstimatedDurationMinutes: duration,
		EstimatedDistanceKm:      distance,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *RideHandler) CancelRide(w http.ResponseWriter, r *http.Request) {
	rideID := r.URL.Query().Get("ride_id")
	if rideID == "" {
		http.Error(w, "ride_id is required", http.StatusBadRequest)
		return
	}

	var req CancelRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.RideService.CancelRide(r.Context(), rideID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
