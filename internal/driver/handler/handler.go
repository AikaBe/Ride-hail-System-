package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/driver/handler/dto"
	"ride-hail/internal/driver/model"
	"ride-hail/internal/driver/service"
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
)

type DriverHandler struct {
	service *service.DriverService
}

func NewHandler(s *service.DriverService) *DriverHandler {
	return &DriverHandler{service: s}
}

func (h *DriverHandler) GetDriverInfo(ctx context.Context, driverID string) (model.DriverInfo, error) {
	driverInfo, err := h.service.GetDriverInfo(ctx, driverID)
	if err != nil {
		return model.DriverInfo{}, err
	}
	return driverInfo, nil
}

func (h *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req dto.OnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	driverSession, err := h.service.GoOnline(ctx, uuid.UUID(driverID), req.Latitude, req.Longitude)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := dto.OnlineResponse{
		Status:    usermodel.DriverStatusAvailable,
		SessionID: string(driverSession.ID),
		Message:   "You are now online and ready to accept rides",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *DriverHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	session, durationHours, err := h.service.GoOffline(ctx, uuid.UUID(driverID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := dto.OfflineResponse{
		Status:    "OFFLINE",
		SessionID: string(session.ID),
		SessionSummary: dto.SessionSummary{
			DurationHours:  durationHours,
			RidesCompleted: session.TotalRides,
			Earnings:       session.TotalEarnings,
		},
		Message: "You are now offline",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req dto.LocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.LocationHistory{
		DriverID:       uuid.UUID(driverID),
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		AccuracyMeters: req.AccuracyMeters,
		SpeedKmh:       req.SpeedKmh,
		HeadingDegrees: req.HeadingDegrees,
	}

	resp, err := h.service.UpdateLocation(ctx, location)
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

func (h *DriverHandler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req dto.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.DriverLocation.Latitude,
		Longitude: req.DriverLocation.Longitude,
	}

	resp, err := h.service.Start(ctx, uuid.UUID(driverID), uuid.UUID(req.RideID), location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DriverHandler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")

	var req dto.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Complete(ctx, uuid.UUID(driverID), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
