package service

import (
	"context"
	"ride-hail/internal/admin/model"
)

type AdminRepository interface {
	GetSystemOverview(ctx context.Context) (*model.SystemOverview, error)
	GetActiveRides(ctx context.Context, page, pageSize int) (*model.ActiveRidesResponse, error)
	GetOnlineDrivers(ctx context.Context) ([]model.OnlineDriver, error)
	GetSystemMetrics(ctx context.Context) (*model.SystemMetrics, error)
}

type AdminService struct {
	repo AdminRepository
}

func NewAdminService(repo AdminRepository) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) GetSystemOverview(ctx context.Context) (*model.SystemOverview, error) {
	overview, err := s.repo.GetSystemOverview(ctx)
	if err != nil {
		return nil, err
	}

	// Add metrics that require calculation
	metrics, err := s.repo.GetSystemMetrics(ctx)
	if err == nil {
		overview.Metrics.AverageWaitTimeMinutes = metrics.AverageWaitTimeMinutes
		overview.Metrics.AverageRideDurationMinutes = metrics.AverageRideDurationMinutes
		overview.Metrics.CancellationRate = metrics.CancellationRate
	}

	return overview, nil
}

func (s *AdminService) GetActiveRides(ctx context.Context, page, pageSize int) (*model.ActiveRidesResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.repo.GetActiveRides(ctx, page, pageSize)
}

func (s *AdminService) GetOnlineDrivers(ctx context.Context) ([]model.OnlineDriver, error) {
	return s.repo.GetOnlineDrivers(ctx)
}

func (s *AdminService) GetSystemMetrics(ctx context.Context) (*model.SystemMetrics, error) {
	return s.repo.GetSystemMetrics(ctx)
}
