package repository

import (
	"context"
	"fmt"
	"ride-hail/internal/admin/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type AdminRepository struct {
	db *pgx.Conn
}

func NewAdminRepository(db *pgx.Conn) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetSystemOverview(ctx context.Context) (*model.SystemOverview, error) {
	overview := &model.SystemOverview{
		Timestamp: time.Now().UTC(),
	}

	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`).Scan(&overview.Metrics.ActiveRides)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rides count: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE'
	`).Scan(&overview.Metrics.AvailableDrivers)
	if err != nil {
		return nil, fmt.Errorf("failed to get available drivers count: %w", err)
	}
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY', 'EN_ROUTE')
	`).Scan(&overview.Metrics.BusyDrivers)
	if err != nil {
		return nil, fmt.Errorf("failed to get busy drivers count: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides WHERE DATE(created_at) = $1
	`, today).Scan(&overview.Metrics.TotalRidesToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's rides count: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0) FROM rides 
		WHERE DATE(created_at) = $1 AND status = 'COMPLETED'
	`, today).Scan(&overview.Metrics.TotalRevenueToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's revenue: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT vehicle_type, COUNT(*) 
		FROM drivers 
		WHERE status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE')
		GROUP BY vehicle_type
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver distribution: %w", err)
	}
	defer rows.Close()

	overview.DriverDistribution = make(map[string]int)
	for rows.Next() {
		var vehicleType string
		var count int
		if err := rows.Scan(&vehicleType, &count); err != nil {
			return nil, err
		}
		overview.DriverDistribution[vehicleType] = count
	}

	rows, err = r.db.Query(ctx, `
		SELECT c.address, COUNT(r.id) as active_rides,
			   (SELECT COUNT(*) FROM drivers d 
				JOIN coordinates cd ON cd.entity_id = d.id AND cd.entity_type = 'driver'
				WHERE cd.address = c.address AND d.status = 'AVAILABLE') as waiting_drivers
		FROM rides r
		JOIN coordinates c ON r.pickup_coordinate_id = c.id
		WHERE r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE')
		GROUP BY c.address
		ORDER BY active_rides DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var hotspot model.Hotspot
		if err := rows.Scan(&hotspot.Location, &hotspot.ActiveRides, &hotspot.WaitingDrivers); err != nil {
			return nil, err
		}
		overview.Hotspots = append(overview.Hotspots, hotspot)
	}

	return overview, nil
}

func (r *AdminRepository) GetActiveRides(ctx context.Context, page, pageSize int) (*model.ActiveRidesResponse, error) {
	response := &model.ActiveRidesResponse{
		Page:     page,
		PageSize: pageSize,
		Rides:    []model.ActiveRide{},
	}

	offset := (page - 1) * pageSize

	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`).Scan(&response.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rides count: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT 
			r.id, r.ride_number, r.status, r.passenger_id, r.driver_id,
			pickup.address as pickup_address, dest.address as destination_address,
			r.started_at, r.completed_at,
			cd.latitude as driver_lat, cd.longitude as driver_lng
		FROM rides r
		LEFT JOIN coordinates pickup ON r.pickup_coordinate_id = pickup.id
		LEFT JOIN coordinates dest ON r.destination_coordinate_id = dest.id
		LEFT JOIN coordinates cd ON cd.entity_id = r.driver_id AND cd.entity_type = 'driver' AND cd.is_current = true
		WHERE r.status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ride model.ActiveRide
		var driverLat, driverLng *float64
		var startedAt, completedAt *time.Time

		err := rows.Scan(
			&ride.RideID, &ride.RideNumber, &ride.Status, &ride.PassengerID, &ride.DriverID,
			&ride.PickupAddress, &ride.DestinationAddress,
			&startedAt, &completedAt,
			&driverLat, &driverLng,
		)
		if err != nil {
			return nil, err
		}

		if startedAt != nil {
			ride.StartedAt = startedAt.Format(time.RFC3339)
		}
		if completedAt != nil {
			ride.CompletedAt = completedAt.Format(time.RFC3339)
		}
		if driverLat != nil && driverLng != nil {
			ride.CurrentDriverLocation = &model.Location{
				Latitude:  *driverLat,
				Longitude: *driverLng,
			}
		}

		response.Rides = append(response.Rides, ride)
	}

	return response, nil
}

func (r *AdminRepository) GetOnlineDrivers(ctx context.Context) ([]model.OnlineDriver, error) {
	drivers := make([]model.OnlineDriver, 0) // Ensure empty array, not nil

	rows, err := r.db.Query(ctx, `
		SELECT 
			d.id, u.email, d.rating, d.status, d.vehicle_type,
			c.latitude, c.longitude, c.address,
			d.total_rides, d.total_earnings
		FROM drivers d
		JOIN users u ON d.id = u.id
		LEFT JOIN coordinates c ON c.entity_id = d.id AND c.entity_type = 'driver' AND c.is_current = true
		WHERE d.status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE')
		ORDER BY d.status, d.rating DESC
	`)
	if err != nil {
		return drivers, fmt.Errorf("failed to get online drivers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var driver model.OnlineDriver
		var lat, lng *float64
		var address *string

		err := rows.Scan(
			&driver.DriverID, &driver.Email, &driver.Rating, &driver.Status, &driver.VehicleType,
			&lat, &lng, &address,
			&driver.TotalRides, &driver.TotalEarnings,
		)
		if err != nil {
			return drivers, err
		}

		if lat != nil && lng != nil {
			driver.CurrentLocation = &model.Location{
				Latitude:  *lat,
				Longitude: *lng,
			}
		}
		if address != nil {
			driver.CurrentAddress = *address
		}

		drivers = append(drivers, driver)
	}

	return drivers, nil
}

func (r *AdminRepository) GetSystemMetrics(ctx context.Context) (*model.SystemMetrics, error) {
	metrics := &model.SystemMetrics{}

	overview, err := r.GetSystemOverview(ctx)
	if err != nil {
		return metrics, nil
	}

	metrics.ActiveRides = overview.Metrics.ActiveRides
	metrics.AvailableDrivers = overview.Metrics.AvailableDrivers
	metrics.BusyDrivers = overview.Metrics.BusyDrivers
	metrics.TotalRidesToday = overview.Metrics.TotalRidesToday
	metrics.TotalRevenueToday = overview.Metrics.TotalRevenueToday

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at)) / 60), 0)
		FROM rides 
		WHERE matched_at IS NOT NULL AND requested_at IS NOT NULL
		AND created_at >= NOW() - INTERVAL '1 day'
	`).Scan(&metrics.AverageWaitTimeMinutes)
	if err != nil {
		metrics.AverageWaitTimeMinutes = 0
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) / 60), 0)
		FROM rides 
		WHERE completed_at IS NOT NULL AND started_at IS NOT NULL
		AND created_at >= NOW() - INTERVAL '1 day'
	`).Scan(&metrics.AverageRideDurationMinutes)
	if err != nil {
		metrics.AverageRideDurationMinutes = 0
	}

	err = r.db.QueryRow(ctx, `
		SELECT 
			COALESCE(
				SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 
				0
			)
		FROM rides 
		WHERE created_at >= NOW() - INTERVAL '1 day'
	`).Scan(&metrics.CancellationRate)
	if err != nil {
		metrics.CancellationRate = 0
	}

	return metrics, nil
}
