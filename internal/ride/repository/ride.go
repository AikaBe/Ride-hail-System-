package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/ride/model"
	"ride-hail/pkg/uuid"

	"github.com/jackc/pgx/v5"
)

type RideRepository struct {
	DB *pgx.Conn
}

func NewRideRepository(database *pgx.Conn) *RideRepository {
	return &RideRepository{DB: database}
}

func (r *RideRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (r *RideRepository) GetPassengerIDByRideID(ctx context.Context, rideID string) (string, error) {
	var passengerID string

	query := `
		SELECT passenger_id
		FROM rides
		WHERE id = $1
	`

	err := r.DB.QueryRow(ctx, query, rideID).Scan(&passengerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("ride with id %s not found", rideID)
		}
		return "", fmt.Errorf("failed to get passenger_id: %w", err)
	}

	return passengerID, nil
}

func (r *RideRepository) UpdateRideStatusMatched(ctx context.Context, rideID string, driverID string) error {
	query := `
		UPDATE rides
		SET 
			status = 'MATCHED',
			driver_id = $1,
			matched_at = $2,
			updated_at = $2
		WHERE id = $3;
	`

	_, err := r.DB.Exec(ctx, query, driverID, time.Now().UTC(), rideID)
	if err != nil {
		return err
	}

	return nil
}

func (r *RideRepository) InsertRide(ctx context.Context, tx pgx.Tx, ride model.Ride) (*model.Ride, error) {
	if tx == nil {
		return &model.Ride{}, fmt.Errorf("transaction is nil")
	}

	query := `
		INSERT INTO rides (
			passenger_id,
			pickup_coordinate_id,
			destination_coordinate_id,
			status,
			ride_number,
			estimated_fare,
			priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	row := tx.QueryRow(ctx, query,
		ride.PassengerID,
		ride.PickupCoordinateID,
		ride.DestinationCoordinateID,
		ride.Status,
		ride.RideNumber,
		ride.EstimatedFare,
		ride.Priority,
	)

	var id string
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan inserted ride: %w", err)
	}

	ride.ID = uuid.UUID(id)
	ride.CreatedAt = createdAt
	ride.UpdatedAt = updatedAt

	return &ride, nil
}

func (r *RideRepository) InsertRideEvent(ctx context.Context, tx pgx.Tx, event model.RideEvent) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	query := `
		INSERT INTO ride_events (ride_id, event_type, event_data)
		VALUES ($1, $2, $3)
	`

	data, _ := json.Marshal(event.EventData)
	if _, err := tx.Exec(ctx, query, event.RideID, event.EventType, data); err != nil {
		return fmt.Errorf("failed to insert ride event: %w", err)
	}

	return nil
}

func (r *RideRepository) InsertCoordinate(ctx context.Context, tx pgx.Tx, coordinate model.Coordinate) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("transaction is nil")
	}

	query := `
		INSERT INTO coordinates (
			entity_id, entity_type, address, latitude, longitude, is_current
		)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id;
	`

	var id string
	err := tx.QueryRow(ctx, query,
		coordinate.EntityID,
		coordinate.EntityType,
		coordinate.Address,
		coordinate.Latitude,
		coordinate.Longitude,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert coordinate: %w", err)
	}

	return id, nil
}

type CancelRideResponse struct {
	RideID      string `json:"ride_id"`
	Status      string `json:"status"`
	CancelledAt string `json:"cancelled_at"`
	Message     string `json:"message"`
}

func (r *RideRepository) CancelRide(ctx context.Context, rideID, reason string) (*CancelRideResponse, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE rides
		SET status = 'CANCELLED', cancelled_at = NOW(), cancellation_reason = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'REQUESTED'
		RETURNING cancelled_at;
	`

	var cancelledAt time.Time
	err = tx.QueryRow(ctx, query, reason, rideID).Scan(&cancelledAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ride not found or cannot be cancelled")
		}
		return nil, fmt.Errorf("failed to cancel ride: %w", err)
	}

	// Fix: Use proper JSON construction for event_data
	eventData := map[string]string{"reason": reason}
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Insert cancellation event
	eventQuery := `
		INSERT INTO ride_events (ride_id, event_type, event_data, created_at)
		VALUES ($1, 'RIDE_CANCELLED', $2, NOW())
	`
	_, err = tx.Exec(ctx, eventQuery, rideID, jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to insert cancellation event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CancelRideResponse{
		RideID:      rideID,
		Status:      "CANCELLED",
		CancelledAt: cancelledAt.Format(time.RFC3339),
		Message:     "Ride cancelled successfully",
	}, nil
}
