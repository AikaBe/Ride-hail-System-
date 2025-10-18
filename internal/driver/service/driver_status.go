package service

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/common/model"
	"time"
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
	if lat < -90 || lat > 90 {
		return model.OnlineResponse{}, errors.New("latitude out of range")
	}
	if lon < -180 || lon > 180 {
		return model.OnlineResponse{}, errors.New("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.OnlineResponse{}, err
	}
	if driverStatus != "OFFLINE" {
		return model.OnlineResponse{}, errors.New("driver is not offline")
	}
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.OfflineResponse{}, err
	}
	if driverStatus == "EN_ROUTE" || driverStatus == "BUSY" {
		return model.OfflineResponse{}, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}
	return s.repo.SetOffline(ctx, driverID)
}

func (s *DriverService) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	if req.Latitude < -90 || req.Latitude > 90 {
		return model.LocationResponse{}, errors.New("latitude out of range")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return model.LocationResponse{}, errors.New("longitude out of range")
	}
	if req.AccuracyMeters > 50 || req.AccuracyMeters < 0 {
		return model.LocationResponse{}, errors.New("location accuracy too low or less than 0")
	}
	if req.SpeedKmh < 0 || req.SpeedKmh > 490 {
		return model.LocationResponse{}, errors.New("invalid speed ")
	}
	if req.HeadingDegrees < 0 || req.HeadingDegrees > 360 {
		return model.LocationResponse{}, errors.New("invalid heading")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.LocationResponse{}, err
	}
	if driverStatus == "OFFLINE" {
		return model.LocationResponse{}, errors.New("driver is OFFLINE")
	}
	return s.repo.Location(ctx, driverID, req)
}

func (s *DriverService) Start(ctx context.Context, driverID string, rideId string, location model.Location) (model.StartResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, rideId)
	if err != nil {
		return model.StartResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		return model.StartResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	if location.Latitude < -90 || location.Latitude > 90 {
		return model.StartResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return model.StartResponse{}, fmt.Errorf("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.StartResponse{}, err
	}
	if driverStatus != "AVAILABLE" {
		return model.StartResponse{}, errors.New("driver is not available")
	}
	return s.repo.Start(ctx, driverID, rideId, location)
}

func (s *DriverService) Complete(ctx context.Context, driverID string, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, req.RideID)
	if err != nil {
		return model.CompleteResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		return model.CompleteResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	const baseFare = 400.0
	const perKmRate = 120.0
	const perMinuteRate = 20.0

	driverEarnings := baseFare +
		req.ActualDistanceKm*perKmRate +
		req.ActualDurationMins*perMinuteRate

	if location.Latitude < -90 || location.Latitude > 90 {
		return model.CompleteResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return model.CompleteResponse{}, fmt.Errorf("longitude out of range")
	}
	if req.ActualDurationMins <= 0 {
		return model.CompleteResponse{}, fmt.Errorf("duration out of range")
	}
	if req.ActualDistanceKm <= 0 {
		return model.CompleteResponse{}, fmt.Errorf("duration out of range")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.CompleteResponse{}, err
	}
	if driverStatus != "BUSY" {
		return model.CompleteResponse{}, errors.New("driver status not busy")
	}
	resp, err := s.repo.Complete(ctx, driverID, driverEarnings, req, location)
	if err != nil {
		return model.CompleteResponse{}, err
	}

	resp.DriverEarning = driverEarnings
	resp.Message = fmt.Sprintf("Ride completed successfully at %s", time.Now().Format(time.RFC3339))

	return resp, nil
}
