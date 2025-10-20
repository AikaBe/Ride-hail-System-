package service

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/model"
)

type DriverRepository interface {
	SetOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error)
	SetOffline(ctx context.Context, driverID string) (model.OfflineResponse, error)
	Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error)
	Start(ctx context.Context, driverID string, rideID string, req model.Location) (model.StartResponse, error)
	Complete(ctx context.Context, driverID string, driverEarning float64, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error)
	GetRideStatus(ctx context.Context, driverID, rideID string) (string, error)
	GetDriverStatus(ctx context.Context, driverID string) (string, error)
}

type DriverService struct {
	repo DriverRepository
}

func NewDriverService(repo DriverRepository) *DriverService {
	return &DriverService{repo: repo}
}

func (s *DriverService) GoOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error) {
	const action = "GoOnline"
	requestID := "" // можно прокинуть реальный requestID если есть

	if lat < -90 || lat > 90 {
		logger.Error(action, "latitude out of range", requestID, "", fmt.Sprintf("%f", lat), "")
		return model.OnlineResponse{}, errors.New("latitude out of range")
	}
	if lon < -180 || lon > 180 {
		logger.Error(action, "longitude out of range", requestID, "", fmt.Sprintf("%f", lon), "")
		return model.OnlineResponse{}, errors.New("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error(action, "cannot get driver status", requestID, "", err.Error(), "")
		return model.OnlineResponse{}, err
	}
	if driverStatus != "OFFLINE" {
		logger.Warn(action, "driver is not offline", requestID, "", driverStatus)
		return model.OnlineResponse{}, errors.New("driver is not offline")
	}
	resp, err := s.repo.SetOnline(ctx, driverID, lat, lon)
	if err != nil {
		logger.Error(action, "repository error", requestID, "", err.Error(), "")
		return resp, err
	}
	logger.Info(action, "driver is now ONLINE", requestID, "")
	return resp, nil
}

func (s *DriverService) GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	const action = "GoOffline"
	requestID := ""

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error(action, "cannot get driver status", requestID, "", err.Error(), "")
		return model.OfflineResponse{}, err
	}
	if driverStatus == "EN_ROUTE" || driverStatus == "BUSY" {
		logger.Warn(action, "driver cannot go offline", requestID, "", driverStatus)
		return model.OfflineResponse{}, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}
	resp, err := s.repo.SetOffline(ctx, driverID)
	if err != nil {
		logger.Error(action, "repository error", requestID, "", err.Error(), "")
		return resp, err
	}
	logger.Info(action, "driver is now OFFLINE", requestID, "")
	return resp, nil
}

func (s *DriverService) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	const action = "Location"
	requestID := ""

	if req.Latitude < -90 || req.Latitude > 90 {
		logger.Error(action, "latitude out of range", requestID, "", fmt.Sprintf("%f", req.Latitude), "")
		return model.LocationResponse{}, errors.New("latitude out of range")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		logger.Error(action, "longitude out of range", requestID, "", fmt.Sprintf("%f", req.Longitude), "")
		return model.LocationResponse{}, errors.New("longitude out of range")
	}
	if req.AccuracyMeters > 50 || req.AccuracyMeters < 0 {
		logger.Warn(action, "invalid accuracy", requestID, "", fmt.Sprintf("%f", req.AccuracyMeters))
		return model.LocationResponse{}, errors.New("location accuracy too low or less than 0")
	}
	if req.SpeedKmh < 0 || req.SpeedKmh > 490 {
		logger.Warn(action, "invalid speed", requestID, "", fmt.Sprintf("%f", req.SpeedKmh))
		return model.LocationResponse{}, errors.New("invalid speed ")
	}
	if req.HeadingDegrees < 0 || req.HeadingDegrees > 360 {
		logger.Warn(action, "invalid heading", requestID, "", fmt.Sprintf("%f", req.HeadingDegrees))
		return model.LocationResponse{}, errors.New("invalid heading")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error(action, "cannot get driver status", requestID, "", err.Error(), "")
		return model.LocationResponse{}, err
	}
	if driverStatus == "OFFLINE" {
		logger.Warn(action, "driver is OFFLINE", requestID, "", "")
		return model.LocationResponse{}, errors.New("driver is OFFLINE")
	}

	resp, err := s.repo.Location(ctx, driverID, req)
	if err != nil {
		logger.Error(action, "repository error", requestID, "", err.Error(), "")
		return resp, err
	}
	logger.Info(action, "location updated", requestID, "")
	return resp, nil
}

func (s *DriverService) Start(ctx context.Context, driverID string, rideId string, location model.Location) (model.StartResponse, error) {
	const action = "Start"
	requestID := ""

	status, err := s.repo.GetRideStatus(ctx, driverID, rideId)
	if err != nil {
		logger.Error(action, "cannot get ride status", requestID, rideId, err.Error(), "")
		return model.StartResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		logger.Warn(action, "ride already completed or cancelled", requestID, rideId, status)
		return model.StartResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error(action, "cannot get driver status", requestID, rideId, err.Error(), "")
		return model.StartResponse{}, err
	}
	if driverStatus != "AVAILABLE" {
		logger.Warn(action, "driver not available", requestID, rideId, driverStatus)
		return model.StartResponse{}, errors.New("driver is not available")
	}

	resp, err := s.repo.Start(ctx, driverID, rideId, location)
	if err != nil {
		logger.Error(action, "repository error", requestID, rideId, err.Error(), "")
		return resp, err
	}
	logger.Info(action, "ride started", requestID, rideId)
	return resp, nil
}

func (s *DriverService) Complete(ctx context.Context, driverID string, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error) {
	const action = "Complete"
	requestID := ""

	status, err := s.repo.GetRideStatus(ctx, driverID, req.RideID)
	if err != nil {
		logger.Error(action, "cannot get ride status", requestID, req.RideID, err.Error(), "")
		return model.CompleteResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		logger.Warn(action, "ride already completed or cancelled", requestID, req.RideID, status)
		return model.CompleteResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}

	resp, err := s.repo.Complete(ctx, driverID, 0, req, location)
	if err != nil {
		logger.Error(action, "repository error", requestID, req.RideID, err.Error(), "")
		return resp, err
	}
	logger.Info(action, "ride completed", requestID, req.RideID)
	return resp, nil
}
