package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/ride/handler/dto"
	"ride-hail/internal/ride/service"
	"ride-hail/internal/user/jwt"

	usermodel "ride-hail/internal/user/model"
)

type RideHandler struct {
	RideService *service.RideService
	jwtManager  *jwt.Manager
}

func NewRideHandler(service *service.RideService, manager *jwt.Manager) *RideHandler {
	return &RideHandler{RideService: service, jwtManager: manager}
}

func (h *RideHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
	const action = "CreateRide"
	requestID := r.Header.Get("X-Request-ID") // если есть requestID из заголовка

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.Role != string(usermodel.RolePassenger) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	var req dto.RideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(action, "invalid JSON in request body", requestID, "", err.Error())
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	ride, pickup, destination, err := dto.MapRideRequestToEntities(req)
	if err != nil {
		logger.Error(action, "failed to map ride request to entities", requestID, "", err.Error())
		http.Error(w, "invalid request mapping", http.StatusBadRequest)
		return
	}

	createdRide, distance, duration, err := h.RideService.CreateRide(r.Context(), ride, pickup, destination)
	if err != nil {
		logger.Error(action, "failed to create ride in service", requestID, "", err.Error())
		http.Error(w, fmt.Sprintf("failed to create ride: %v", err), http.StatusInternalServerError)
		return
	}

	resp := dto.RideResponse{
		RideID:                   string(createdRide.ID),
		RideNumber:               createdRide.RideNumber,
		Status:                   string(*createdRide.Status),
		EstimatedFare:            *createdRide.EstimatedFare,
		EstimatedDurationMinutes: duration,
		EstimatedDistanceKm:      distance,
	}

	logger.Info(action, "ride created successfully", requestID, string(createdRide.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error(action, "failed to encode response", requestID, string(createdRide.ID), err.Error())
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
func (h *RideHandler) CancelRide(w http.ResponseWriter, r *http.Request) {
	const action = "CancelRide"
	requestID := r.Header.Get("X-Request-ID")

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.Role != string(usermodel.RolePassenger) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	rideID := r.PathValue("ride_id")
	if rideID == "" {
		logger.Warn(action, "ride_id not provided in path", requestID, "", "")
		http.Error(w, "ride_id is required", http.StatusBadRequest)
		return
	}

	var req dto.CancelRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(action, "invalid JSON in request body", requestID, rideID, err.Error())
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.RideService.CancelRide(r.Context(), rideID, req.Reason)
	if err != nil {
		logger.Error(action, "failed to cancel ride", requestID, rideID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info(action, "ride cancelled successfully", requestID, rideID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
