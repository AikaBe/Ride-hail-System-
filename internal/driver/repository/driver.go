package repository

import (
	"context"
	"ride-hail/internal/common/logger"
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

// Set driver online
func (r *DriverRepository) SetOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("set_online_request", "Driver going online", requestID, driverID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		logger.Error("tx_start_failed", "Failed to start transaction", requestID, driverID, err.Error(), "")
		return model.OnlineResponse{}, err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id, started_at)
		VALUES ($1, now())
		RETURNING id
	`, driverID).Scan(&sessionID)
	if err != nil {
		logger.Error("insert_session_failed", "Failed to insert driver session", requestID, driverID, err.Error(), "")
		return model.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		logger.Error("update_driver_failed", "Failed to update driver status", requestID, driverID, err.Error(), "")
		return model.OnlineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (driver_id, latitude, longitude)
		VALUES ($1, $2, $3)
	`, driverID, lat, lon)
	if err != nil {
		logger.Error("insert_location_failed", "Failed to insert location history", requestID, driverID, err.Error(), "")
		return model.OnlineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "Failed to commit transaction", requestID, driverID, err.Error(), "")
		return model.OnlineResponse{}, err
	}

	logger.Info("set_online_success", "Driver is now online", requestID, driverID)
	return model.OnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}, nil
}

// Set driver offline
func (r *DriverRepository) SetOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("set_offline_request", "Driver going offline", requestID, driverID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		logger.Error("tx_start_failed", "Failed to start transaction", requestID, driverID, err.Error(), "")
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
		logger.Error("session_not_found", "Active session not found", requestID, driverID, err.Error(), "")
		return model.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE id = $1
	`, sessionID)
	if err != nil {
		logger.Error("update_session_failed", "Failed to update driver session", requestID, driverID, err.Error(), "")
		return model.OfflineResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		logger.Error("update_driver_failed", "Failed to update driver status", requestID, driverID, err.Error(), "")
		return model.OfflineResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "Failed to commit transaction", requestID, driverID, err.Error(), "")
		return model.OfflineResponse{}, err
	}

	durationHours := time.Since(startedAt).Hours()
	logger.Info("set_offline_success", "Driver is now offline", requestID, driverID)

	return model.OfflineResponse{
		Status:    "OFFLINE",
		SessionID: sessionID,
		SessionSummary: model.SessionSummary{
			DurationHours:  durationHours,
			RidesCompleted: totalRides,
			Earnings:       totalEarnings,
		},
		Message: "You are now offline",
	}, nil
}

// Update driver location
func (r *DriverRepository) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("location_update_request", "Updating driver location", requestID, driverID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		logger.Error("tx_start_failed", "Failed to start transaction", requestID, driverID, err.Error(), "")
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
				logger.Error("insert_coordinate_failed", "Failed to insert coordinate", requestID, driverID, err.Error(), "")
				return model.LocationResponse{}, err
			}
		} else {
			logger.Error("query_coordinate_failed", "Failed to get current coordinate", requestID, driverID, err.Error(), "")
			return model.LocationResponse{}, err
		}
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE coordinates
			SET latitude = $1, longitude = $2, updated_at = now()
			WHERE id = $3
		`, req.Latitude, req.Longitude, coordinateID)
		if err != nil {
			logger.Error("update_coordinate_failed", "Failed to update coordinate", requestID, driverID, err.Error(), "")
			return model.LocationResponse{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, coordinateID, driverID, req.Latitude, req.Longitude, req.AccuracyMeters, req.SpeedKmh, req.HeadingDegrees)
	if err != nil {
		logger.Error("insert_location_history_failed", "Failed to insert location history", requestID, driverID, err.Error(), "")
		return model.LocationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "Failed to commit transaction", requestID, driverID, err.Error(), "")
		return model.LocationResponse{}, err
	}

	logger.Info("location_update_success", "Driver location updated", requestID, driverID)
	return model.LocationResponse{
		CoordinateID: coordinateID,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Start ride
func (r *DriverRepository) Start(ctx context.Context, driverID, rideID string, req model.Location) (model.StartResponse, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("ride_start_request", "Starting ride", requestID, driverID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		logger.Error("tx_start_failed", "Failed to start transaction", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE rides
		SET status = 'IN_PROGRESS', started_at = now(), updated_at = now()
		WHERE id = $1
	`, rideID)
	if err != nil {
		logger.Error("update_ride_failed", "Failed to update ride status", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'BUSY', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		logger.Error("update_driver_failed", "Failed to update driver status", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
		VALUES ($1, 'driver', 'current location', $2, $3, true)
	`, driverID, req.Latitude, req.Longitude)
	if err != nil {
		logger.Error("insert_coordinate_failed", "Failed to insert coordinate", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES ($1, 'RIDE_STARTED', jsonb_build_object('driver_id', $2, 'location', jsonb_build_object('latitude', $3, 'longitude', $4), 'started_at', now()))
	`, rideID, driverID, req.Latitude, req.Longitude)
	if err != nil {
		logger.Error("insert_ride_event_failed", "Failed to insert ride event", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "Failed to commit transaction", requestID, driverID, err.Error(), "")
		return model.StartResponse{}, err
	}

	logger.Info("ride_start_success", "Ride started successfully", requestID, driverID)
	return model.StartResponse{
		RideID:    rideID,
		Status:    "BUSY",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Message:   "Ride started successfully",
	}, nil
}

// Complete ride
func (r *DriverRepository) Complete(ctx context.Context, driverID string, driverEarning float64, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("ride_complete_request", "Completing ride", requestID, driverID)

	var resp model.CompleteResponse
	tx, err := r.db.Begin(ctx)
	if err != nil {
		logger.Error("tx_start_failed", "Failed to start transaction", requestID, driverID, err.Error(), "")
		return resp, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		UPDATE rides
		SET status = 'COMPLETED', final_fare = $1, final_latitude = $2, final_longitude = $3,
			actual_distance_km = $4, actual_duration_min = $5, completed_at = now()
		WHERE id = $6 AND driver_id = $7
		RETURNING id, status, to_char(completed_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, driverEarning, location.Latitude, location.Longitude, req.ActualDistanceKm, req.ActualDurationMins, req.RideID, driverID).Scan(&resp.RideID, &resp.Status, &resp.CompletedAt)

	if err != nil {
		logger.Error("update_ride_failed", "Failed to update ride", requestID, driverID, err.Error(), "")
		return resp, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET total_rides = total_rides + 1, total_earnings = total_earnings + $1, status = 'AVAILABLE', updated_at = now()
		WHERE id = $2
	`, driverEarning, driverID)
	if err != nil {
		logger.Error("update_driver_failed", "Failed to update driver stats", requestID, driverID, err.Error(), "")
		return resp, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES ($1, 'RIDE_COMPLETED', jsonb_build_object('driver_id', $2, 'earned', $3, 'distance_km', $4, 'duration_min', $5, 'completed_at', now()))
	`, req.RideID, driverID, driverEarning, req.ActualDistanceKm, req.ActualDurationMins)
	if err != nil {
		logger.Error("insert_ride_event_failed", "Failed to insert ride event", requestID, driverID, err.Error(), "")
		return resp, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "Failed to commit transaction", requestID, driverID, err.Error(), "")
		return resp, err
	}

	logger.Info("ride_complete_success", "Ride completed successfully", requestID, driverID)
	return resp, nil
}

// Get ride status
func (r *DriverRepository) GetRideStatus(ctx context.Context, driverID, rideID string) (string, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("get_ride_status_request", "Fetching ride status", requestID, driverID)

	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status FROM rides WHERE id = $1 AND driver_id = $2
	`, rideID, driverID).Scan(&status)
	if err != nil {
		logger.Error("get_ride_status_failed", "Failed to get ride status", requestID, driverID, err.Error(), "")
		return "", err
	}

	logger.Info("get_ride_status_success", "Ride status fetched", requestID, driverID)
	return status, nil
}

// Get driver status
func (r *DriverRepository) GetDriverStatus(ctx context.Context, driverID string) (string, error) {
	requestID := ctx.Value("request_id").(string)
	logger.Info("get_driver_status_request", "Fetching driver status", requestID, driverID)

	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status FROM drivers WHERE id = $1
	`, driverID).Scan(&status)
	if err != nil {
		logger.Error("get_driver_status_failed", "Failed to get driver status", requestID, driverID, err.Error(), "")
		return "", err
	}

	logger.Info("get_driver_status_success", "Driver status fetched", requestID, driverID)
	return status, nil
}
