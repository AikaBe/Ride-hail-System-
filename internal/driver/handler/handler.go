package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"ride-hail-system/internal/common/logger"
	"ride-hail-system/internal/driver/handler/dto"
	"ride-hail-system/internal/driver/model"
	"ride-hail-system/internal/driver/service"
	"ride-hail-system/internal/user/jwt"
	usermodel "ride-hail-system/internal/user/model"
	"ride-hail-system/pkg/uuid"
)

type DriverHandler struct {
	service    *service.DriverService
	jwtManager *jwt.Manager
}

func NewHandler(s *service.DriverService, jwtManager *jwt.Manager) *DriverHandler {
	return &DriverHandler{service: s, jwtManager: jwtManager}
}

func (h *DriverHandler) GetDriverInfo(ctx context.Context, driverID string) (model.DriverInfo, error) {
	logger.Info("get_driver_info", "Fetching driver info", "", driverID)
	driverInfo, err := h.service.GetDriverInfo(ctx, driverID)
	if err != nil {
		logger.Error("get_driver_info", "Failed to get driver info", "", driverID, err.Error())
		return model.DriverInfo{}, err
	}
	logger.Info("get_driver_info", "Driver info fetched successfully", "", driverID)
	return driverInfo, nil
}

func (h *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	logger.Info("go_online", "Driver attempting to go online", "", driverID)

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if strings.TrimSpace(claims.UserID) != strings.TrimSpace(driverID) {
		http.Error(w, "forbidden: token does not match driver", http.StatusForbidden)
		return
	}
	if strings.TrimSpace(claims.Role) != strings.TrimSpace(string(usermodel.RoleDriver)) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	var req dto.OnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("go_online", "Invalid request body", "", driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	driverSession, err := h.service.GoOnline(ctx, uuid.UUID(driverID), req.Latitude, req.Longitude)
	if err != nil {
		logger.Error("go_online", "Failed to set driver online", "", driverID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("go_online", "Driver is now online", "", driverID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := dto.OnlineResponse{
		Status:    usermodel.DriverStatusAvailable,
		SessionID: string(driverSession.ID),
		Message:   "You are now online and ready to accept rides",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("go_online", "Failed to encode response", "", driverID, err.Error())
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *DriverHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	logger.Info("go_offline", "Driver attempting to go offline", "", driverID)

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.UserID != driverID {
		http.Error(w, "forbidden: token does not match driver", http.StatusForbidden)
		return
	}
	if claims.Role != string(usermodel.RoleDriver) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	session, durationHours, err := h.service.GoOffline(ctx, uuid.UUID(driverID))
	if err != nil {
		logger.Error("go_offline", "Failed to set driver offline", "", driverID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("go_offline", "Driver is now offline", "", driverID)
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
		logger.Error("go_offline", "Failed to encode response", "", driverID, err.Error())
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	logger.Debug("update_location", "Driver updating location", "", driverID)

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.UserID != driverID {
		http.Error(w, "forbidden: token does not match driver", http.StatusForbidden)
		return
	}
	if claims.Role != string(usermodel.RoleDriver) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	var req dto.LocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("update_location", "Invalid request body", "", driverID, err.Error())
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
		logger.Error("update_location", "Failed to update driver location", "", driverID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("update_location", "Driver location updated successfully", "", driverID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("update_location", "Failed to encode response", "", driverID, err.Error())
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *DriverHandler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	logger.Info("start_ride", "Driver starting ride", "", driverID)

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.UserID != driverID {
		http.Error(w, "forbidden: token does not match driver", http.StatusForbidden)
		return
	}
	if claims.Role != string(usermodel.RoleDriver) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	var req dto.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("start_ride", "Invalid request body", "", driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	location := model.Location{
		Latitude:  req.DriverLocation.Latitude,
		Longitude: req.DriverLocation.Longitude,
	}

	resp, err := h.service.Start(ctx, uuid.UUID(driverID), uuid.UUID(req.RideID), location)
	if err != nil {
		logger.Error("start_ride", "Failed to start ride", "", driverID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("start_ride", "Ride started successfully", "", driverID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DriverHandler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	driverID := r.PathValue("driver_id")
	logger.Info("complete_ride", "Driver completing ride", "", driverID)

	claims, err := h.jwtManager.ExtractClaims(w, r)

	if claims.UserID != driverID {
		http.Error(w, "forbidden: token does not match driver", http.StatusForbidden)
		return
	}
	if claims.Role != string(usermodel.RoleDriver) {
		http.Error(w, "forbidden: not authorized", http.StatusUnauthorized)
		return
	}

	var req dto.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("complete_ride", "Invalid request body", "", driverID, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Complete(ctx, uuid.UUID(driverID), req)
	if err != nil {
		logger.Error("complete_ride", "Failed to complete ride", "", driverID, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("complete_ride", "Ride completed successfully", "", driverID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
