package repository

import (
	"context"
	"ride-hail/internal/common/models"
	"time"

	"github.com/jackc/pgx/v5"
)

type DriverRepository struct {
	db *pgx.Conn
}

func NewDriverRepository(db *pgx.Conn) *DriverRepository {
	return &DriverRepository{db: db}
}

func (r *DriverRepository) SetOnline(ctx context.Context, driverID string, lat, lon float64) (models.OnlineResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.OnlineResponse{}, err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id,started_at = now())
		VALUES ($1)
		RETURNING id
	`, driverID).Scan(&sessionID)
	if err != nil {
		return models.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return models.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history ( driver_id, latitude, longitude)
		VALUES ( $1, $2, $3)
	`, driverID, lat, lon)
	if err != nil {
		return models.OnlineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.OnlineResponse{}, err
	}

	response := models.OnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}

	return response, nil
}

func (r *DriverRepository) SetOffline(ctx context.Context, driverID string) (models.OfflineResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.OfflineResponse{}, err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	var startedAt time.Time
	var totalRides float64
	var totalEarnings float64

	err = tx.QueryRow(ctx, `
		SELECT id, started_at, total_rides, total_earnings
		FROM driver_sessions
		WHERE driver_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`, driverID).Scan(&sessionID, &startedAt, &totalRides, &totalEarnings)
	if err != nil {
		return models.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE id = $1
	`, sessionID)
	if err != nil {
		return models.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return models.OfflineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.OfflineResponse{}, err
	}

	durationHours := time.Since(startedAt).Hours()

	response := models.OfflineResponse{
		Status:    "OFFLINE",
		SessionID: sessionID,
		SessionSummary: models.SessionSummary{
			DurationHours:  durationHours,
			RidesCompleted: totalRides,
			Earnings:       totalEarnings,
		},
		Message: "You are now offline",
	}

	return response, nil
}

func (r *DriverRepository) Location(ctx context.Context, driverID string, req models.LocationRequest) (models.LocationResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.LocationResponse{}, err
	}
	defer tx.Rollback(ctx)

	var coordinateID string

	err = tx.QueryRow(ctx, `
		SELECT id FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
		LIMIT 1
	`, driverID).Scan(&coordinateID)

	if err != nil {
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, `
				INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
				VALUES ($1, 'driver', 'Unknown', $2, $3, true)
				RETURNING id
			`, driverID, req.Latitude, req.Longitude).Scan(&coordinateID)
			if err != nil {
				return models.LocationResponse{}, err
			}
		} else {
			return models.LocationResponse{}, err
		}
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE coordinates
			SET latitude = $1,
				longitude = $2,
				updated_at = now()
			WHERE id = $3
		`, req.Latitude, req.Longitude, coordinateID)
		if err != nil {
			return models.LocationResponse{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (
			coordinate_id, driver_id, latitude, longitude,
			accuracy_meters, speed_kmh, heading_degrees
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, coordinateID, driverID, req.Latitude, req.Longitude,
		req.AccuracyMeters, req.SpeedKmh, req.HeadingDegrees)
	if err != nil {
		return models.LocationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.LocationResponse{}, err
	}

	resp := models.LocationResponse{
		CoordinateID: coordinateID,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	return resp, nil
}

func (r *DriverRepository) Start(ctx context.Context, driverID string, req models.StartRequest) (models.StartResponse, error) {
	var resp models.StartResponse
	startedAt := time.Now().UTC().Format(time.RFC3339)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return resp, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
		VALUES ($1, 'driver', 'Unknown', $2, $3, true)
	`, driverID, req.DriverLocation.Latitude, req.DriverLocation.Longitude)
	if err != nil {
		return resp, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE rides
		SET status = 'IN_PROGRESS', started_at = $2
		WHERE id = $1
	`, req.RideID, startedAt)
	if err != nil {
		return resp, err
	}

	if err = tx.Commit(ctx); err != nil {
		return resp, err
	}

	resp = models.StartResponse{
		RideID:    req.RideID,
		Status:    "IN_PROGRESS",
		StartedAt: startedAt,
		Message:   "Ride started successfully",
	}
	return resp, nil
}
