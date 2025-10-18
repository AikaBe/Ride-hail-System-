package repository

import (
	"context"
	"fmt"
	"ride-hail/internal/common/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type DriverRepository struct {
	db *pgx.Conn
}

func NewDriverRepository(db *pgx.Conn) *DriverRepository {
	return &DriverRepository{db: db}
}

func (r *DriverRepository) SetOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.OnlineResponse{}, err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id,started_at)
		VALUES ($1,now())
		RETURNING id
	`, driverID).Scan(&sessionID)
	if err != nil {
		return model.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return model.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history ( driver_id, latitude, longitude)
		VALUES ( $1, $2, $3)
	`, driverID, lat, lon)
	if err != nil {
		return model.OnlineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.OnlineResponse{}, err
	}

	response := model.OnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}

	return response, nil
}

func (r *DriverRepository) SetOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.OfflineResponse{}, err
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
		return model.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE id = $1
	`, sessionID)
	if err != nil {
		return model.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return model.OfflineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.OfflineResponse{}, err
	}

	durationHours := time.Since(startedAt).Hours()

	response := model.OfflineResponse{
		Status:    "OFFLINE",
		SessionID: sessionID,
		SessionSummary: model.SessionSummary{
			DurationHours:  durationHours,
			RidesCompleted: totalRides,
			Earnings:       totalEarnings,
		},
		Message: "You are now offline",
	}

	return response, nil
}

func (r *DriverRepository) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.LocationResponse{}, err
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
				return model.LocationResponse{}, err
			}
		} else {
			return model.LocationResponse{}, err
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
			return model.LocationResponse{}, err
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
		return model.LocationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.LocationResponse{}, err
	}

	resp := model.LocationResponse{
		CoordinateID: coordinateID,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	return resp, nil
}

func (r *DriverRepository) Start(ctx context.Context, driverID string, rideID string, req model.Location) (model.StartResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.StartResponse{}, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE rides
		SET status = 'IN_PROGRESS',
		    started_at = now(),
		    updated_at = now()
		WHERE id = $1
	`, rideID)
	if err != nil {
		return model.StartResponse{}, fmt.Errorf("failed to update ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'BUSY',
		    updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return model.StartResponse{}, fmt.Errorf("failed to update driver status: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
		VALUES ($1, 'driver', 'current location', $2, $3, true)
	`, driverID, req.Latitude, req.Longitude)
	if err != nil {
		return model.StartResponse{}, fmt.Errorf("failed to insert coordinates: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES (
			$1,
			'RIDE_STARTED',
			jsonb_build_object(
				'driver_id', $2,
				'location', jsonb_build_object('latitude', $3, 'longitude', $4),
				'started_at', now()
			)
		)
	`, rideID, driverID, req.Latitude, req.Longitude)
	if err != nil {
		return model.StartResponse{}, fmt.Errorf("failed to insert ride event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.StartResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	responce := model.StartResponse{
		RideID:    rideID,
		Status:    "BUSY",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Message:   "Ride started successfully",
	}

	return responce, nil
}

func (r *DriverRepository) Complete(ctx context.Context, driverID string, driverEarning float64, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error) {
	var resp model.CompleteResponse

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		UPDATE rides
		SET 
			status = 'COMPLETED',
			final_fare = $1,
			final_latitude = $2,
			final_longitude = $3,
			actual_distance_km = $4,
			actual_duration_min = $5,
			completed_at = now()
		WHERE id = $6 AND driver_id = $7
		RETURNING id, status, to_char(completed_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`,
		driverEarning,
		location.Latitude,
		location.Longitude,
		req.ActualDistanceKm,
		req.ActualDurationMins,
		req.RideID,
		driverID,
	).Scan(&resp.RideID, &resp.Status, &resp.CompletedAt)

	if err != nil {
		return resp, fmt.Errorf("failed to update ride: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET 
			total_rides = total_rides + 1,
			total_earnings = total_earnings + $1,
			status = 'AVAILABLE',
			updated_at = now()
		WHERE id = $2
	`, driverEarning, driverID)
	if err != nil {
		return resp, fmt.Errorf("failed to update driver stats: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES ($1, 'RIDE_COMPLETED', jsonb_build_object(
			'driver_id', $2,
			'earned', $3,
			'distance_km', $4,
			'duration_min', $5,
			'completed_at', now()
		))
	`, req.RideID, driverID, driverEarning, req.ActualDistanceKm, req.ActualDurationMins)
	if err != nil {
		return resp, fmt.Errorf("failed to insert ride event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return resp, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return resp, nil
}

func (r *DriverRepository) GetRideStatus(ctx context.Context, driverID, rideID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status 
		FROM rides 
		WHERE id = $1 AND driver_id = $2
	`, rideID, driverID).Scan(&status)

	if err != nil {
		return "", fmt.Errorf("failed to get ride status: %w", err)
	}
	return status, nil
}

func (r *DriverRepository) GetDriverStatus(ctx context.Context, driverID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status 
		FROM drivers 
		WHERE id = $1
	`, driverID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}
