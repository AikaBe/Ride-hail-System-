package repository

import (
	"context"
	"fmt"
	"ride-hail/internal/driver/model"
	ridemodel "ride-hail/internal/ride/model"
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
	"time"

	"github.com/jackc/pgx/v5"
)

type DriverRepository struct {
	db *pgx.Conn
}

func NewDriverRepository(db *pgx.Conn) *DriverRepository {
	return &DriverRepository{db: db}
}

func (r *DriverRepository) FindNearbyDrivers(ctx context.Context, pickup model.Location, vehicleType usermodel.VehicleType, radiusMeters float64) ([]model.DriverNearby, error) {
	query := `
		SELECT d.id, u.email, d.rating, c.latitude, c.longitude,
		       ST_Distance(
		         ST_MakePoint(c.longitude, c.latitude)::geography,
		         ST_MakePoint($1, $2)::geography
		       ) / 1000 AS distance_km
		FROM drivers d
		JOIN users u ON d.id = u.id
		JOIN coordinates c ON c.entity_id = d.id
		  AND c.entity_type = 'driver'
		  AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
		  AND d.vehicle_type = $3
		  AND ST_DWithin(
		        ST_MakePoint(c.longitude, c.latitude)::geography,
		        ST_MakePoint($1, $2)::geography,
		        $4
		      )
		ORDER BY distance_km, d.rating DESC
		LIMIT 10;
	`

	rows, err := r.db.Query(ctx, query, pickup.Longitude, pickup.Latitude, vehicleType, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var drivers []model.DriverNearby
	for rows.Next() {
		var d model.DriverNearby
		if err := rows.Scan(&d.ID, &d.Email, &d.Rating, &d.Latitude, &d.Longitude, &d.Distance); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		drivers = append(drivers, d)
	}

	return drivers, nil
}

func (r *DriverRepository) SetOnline(ctx context.Context, driverID uuid.UUID, lat, lon float64) (model.DriverSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.DriverSession{}, err
	}
	defer tx.Rollback(ctx)

	var session model.DriverSession
	err = tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id,started_at)
		VALUES ($1,now())
		RETURNING id
	`, driverID).Scan(&session.ID, &session.DriverID, &session.StartedAt)
	if err != nil {
		return model.DriverSession{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'AVAILABLE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return model.DriverSession{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_history ( driver_id, latitude, longitude)
		VALUES ( $1, $2, $3)
	`, driverID, lat, lon)
	if err != nil {
		return model.DriverSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.DriverSession{}, err
	}

	return session, nil
}

func (r *DriverRepository) SetOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.DriverSession{}, err
	}
	defer tx.Rollback(ctx)

	var session model.DriverSession
	err = tx.QueryRow(ctx, `
		SELECT id, started_at, total_rides, total_earnings
		FROM driver_sessions
		WHERE driver_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`, driverID).Scan(&session.ID, &session.DriverID, &session.StartedAt, &session.TotalRides, &session.TotalEarnings)
	if err != nil {
		return model.DriverSession{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE driver_sessions
		SET ended_at = now()
		WHERE id = $1
	`, session.ID)
	if err != nil {
		return model.DriverSession{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'OFFLINE', updated_at = now()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return model.DriverSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.DriverSession{}, err
	}

	endedAt := time.Now()
	session.EndedAt = &endedAt

	return session, nil
}

func (r *DriverRepository) Location(ctx context.Context, location model.LocationHistory) (ridemodel.Coordinate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ridemodel.Coordinate{}, err
	}
	defer tx.Rollback(ctx)

	var coord ridemodel.Coordinate

	err = tx.QueryRow(ctx, `
        SELECT id, entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at
        FROM coordinates
        WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
        LIMIT 1
    `, location.DriverID).Scan(
		&coord.ID, &coord.EntityID, &coord.EntityType, &coord.Address,
		&coord.Latitude, &coord.Longitude, &coord.IsCurrent,
		&coord.CreatedAt, &coord.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, `
                INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
                VALUES ($1, 'driver', 'Unknown', $2, $3, true)
                RETURNING id, entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at
            `, location.DriverID, location.Latitude, location.Longitude).Scan(
				&coord.ID, &coord.EntityID, &coord.EntityType, &coord.Address,
				&coord.Latitude, &coord.Longitude, &coord.IsCurrent,
				&coord.CreatedAt, &coord.UpdatedAt,
			)
			if err != nil {
				return ridemodel.Coordinate{}, err
			}
		} else {
			return ridemodel.Coordinate{}, err
		}
	} else {
		err = tx.QueryRow(ctx, `
            UPDATE coordinates
            SET latitude = $1,
                longitude = $2,
                updated_at = now()
            WHERE id = $3
            RETURNING id, entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at
        `, location.Latitude, location.Longitude, coord.ID).Scan(
			&coord.ID, &coord.EntityID, &coord.EntityType, &coord.Address,
			&coord.Latitude, &coord.Longitude, &coord.IsCurrent,
			&coord.CreatedAt, &coord.UpdatedAt,
		)
		if err != nil {
			return ridemodel.Coordinate{}, err
		}
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO location_history (
            coordinate_id, driver_id, latitude, longitude,
            accuracy_meters, speed_kmh, heading_degrees
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, coord.ID, location.DriverID, location.Latitude, location.Longitude, location.AccuracyMeters, location.SpeedKmh, location.HeadingDegrees)
	if err != nil {
		return ridemodel.Coordinate{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ridemodel.Coordinate{}, err
	}

	return coord, nil
}

func (r *DriverRepository) Start(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, loc model.Location) (usermodel.DriverStatus, time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback(ctx)

	var startedAt time.Time
	err = tx.QueryRow(ctx, `
	UPDATE rides
	SET status = 'IN_PROGRESS',
	    started_at = now(),
	    updated_at = now()
	WHERE id = $1
	RETURNING started_at
`, rideID).Scan(&startedAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to update ride: %w", err)
	}

	newStatus := usermodel.DriverStatus("BUSY")
	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = $2,
		    updated_at = now()
		WHERE id = $1
	`, driverID, newStatus)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to update driver status: %w", err)
	}

	var coordinateID uuid.UUID
	err = tx.QueryRow(ctx, `
        INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
        VALUES ($1, 'driver', 'current location', $2, $3, true)
        RETURNING id
    `, driverID, loc.Latitude, loc.Longitude).Scan(&coordinateID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to insert coordinates: %w", err)
	}

	var history model.LocationHistory
	err = tx.QueryRow(ctx, `
        INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, recorded_at, ride_id)
        VALUES ($1, $2, $3, $4, now(), $5)
        RETURNING id, coordinate_id, driver_id, latitude, longitude, recorded_at, ride_id
    `, coordinateID, driverID, loc.Latitude, loc.Longitude, rideID).Scan(
		&history.ID, &history.CoordinateID, &history.DriverID,
		&history.Latitude, &history.Longitude, &history.RecordedAt, &history.RideID,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to insert location history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", time.Time{}, fmt.Errorf("commit failed: %w", err)
	}

	return newStatus, startedAt, nil
}

func (r *DriverRepository) Complete(ctx context.Context, driverID uuid.UUID, driverEarning float64, location model.Location, distance, duration float64) (time.Time, error) {
	var completedAt time.Time

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to start transaction: %w", err)
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
		WHERE driver_id = $6
		RETURNING completed_at
	`,
		driverEarning,
		location.Latitude,
		location.Longitude,
		distance,
		duration,
		driverID,
	).Scan(&completedAt)

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to update ride: %w", err)
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
		return time.Time{}, fmt.Errorf("failed to update driver stats: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES (
			(SELECT id FROM rides WHERE driver_id = $1 ORDER BY completed_at DESC LIMIT 1),
			'RIDE_COMPLETED',
			jsonb_build_object(
				'driver_id', $1,
				'earned', $2,
				'distance_km', $3,
				'duration_min', $4,
				'completed_at', now()
			)
		)
	`, driverID, driverEarning, distance, duration)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to insert ride event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return completedAt, nil
}

func (r *DriverRepository) GetRideStatus(ctx context.Context, driverID, rideID uuid.UUID) (ridemodel.RideStatus, error) {
	var status ridemodel.RideStatus
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

func (r *DriverRepository) GetDriverStatus(ctx context.Context, driverID uuid.UUID) (usermodel.DriverStatus, error) {
	var status usermodel.DriverStatus
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
