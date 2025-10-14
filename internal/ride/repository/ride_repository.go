package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type RideRepository struct {
	DB *pgx.Conn
}

func NewRideRepository(database *pgx.Conn) *RideRepository {
	return &RideRepository{DB: database}
}

func (r *RideRepository) InsertRide(ctx context.Context, rideNumber, passengerID, rideType string, fare float64, pickupCoordID, destCoordID string) (string, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO rides (ride_number, passenger_id, vehicle_type, status, estimated_fare, pickup_coordinate_id, destination_coordinate_id)
		VALUES ($1, $2, $3, 'REQUESTED', $4, $5, $6)
		RETURNING id;
	`
	var id string
	err = tx.QueryRow(ctx, query, rideNumber, passengerID, rideType, fare, pickupCoordID, destCoordID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert ride: %w", err)
	}

	// Insert ride event for audit trail
	eventQuery := `
		INSERT INTO ride_events (ride_id, event_type, event_data, created_at)
		VALUES ($1, 'REQUESTED', '{}', NOW())
	`
	_, err = tx.Exec(ctx, eventQuery, id)
	if err != nil {
		return "", fmt.Errorf("failed to insert ride event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}

func (r *RideRepository) InsertCoordinate(ctx context.Context, entityID, entityType, address string, latitude, longitude float64) (string, error) {
	query := `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
		RETURNING id;
	`
	var id string
	err := r.DB.QueryRow(ctx, query, entityID, entityType, address, latitude, longitude).Scan(&id)
	return id, err
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

	// Insert cancellation event
	eventQuery := `
		INSERT INTO ride_events (ride_id, event_type, event_data, created_at)
		VALUES ($1, 'CANCELLED', json_build_object('reason', $2), NOW())
	`
	_, err = tx.Exec(ctx, eventQuery, rideID, reason)
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
