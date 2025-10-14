package service

import (
	"context"
	"ride-hail/internal/common/models"
	"ride-hail/internal/driver/repository"
)

type DriverService struct {
	repo *repository.DriverRepository
}

func NewDriverService(repo *repository.DriverRepository) *DriverService {
	return &DriverService{repo: repo}
}

func (s *DriverService) GoOnline(ctx context.Context, driverID string, lat, lon float64) (models.OnlineResponse, error) {
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID string) (models.OfflineResponse, error) {
	return s.repo.SetOffline(ctx, driverID)
}

func (s *DriverService) Location(ctx context.Context, driverID string, req models.LocationRequest) (models.LocationResponse, error) {
	return s.repo.Location(ctx, driverID, req)
}

func (s *DriverService) Start(ctx context.Context, driverID string, req models.StartRequest) (models.StartResponse, error) {
	return s.repo.Start(ctx, driverID, req)
}
