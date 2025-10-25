package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"ride-hail/internal/user/model"
)

type UserRepository struct {
	db *pgx.Conn
}

func NewUserRepository(db *pgx.Conn) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *UserRepository) CreateUser(ctx context.Context, tx pgx.Tx, user model.User) (model.User, error) {
	var created model.User

	query := `
		INSERT INTO users (email, role, status, password_hash, attrs)
		VALUES ($1, $2, COALESCE($3, 'ACTIVE'), $4, $5)
		RETURNING id, created_at, updated_at, email, role, status, password_hash, attrs
	`

	err := tx.QueryRow(
		ctx,
		query,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
		user.Attrs,
	).Scan(
		&created.ID,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Email,
		&created.Role,
		&created.Status,
		&created.PasswordHash,
		&created.Attrs,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	return created, nil
}

func (r *UserRepository) CreateDriver(ctx context.Context, tx pgx.Tx, driver model.Driver) (model.Driver, error) {
	var created model.Driver
	var vehicleAttrsJSON []byte

	if driver.VehicleAttrs != nil {
		b, err := json.Marshal(driver.VehicleAttrs)
		if err != nil {
			return model.Driver{}, fmt.Errorf("failed to marshal vehicle_attrs: %w", err)
		}
		vehicleAttrsJSON = b
	}

	query := `
		INSERT INTO drivers (
			id, license_number, vehicle_type, vehicle_attrs, rating, total_rides,
			total_earnings, status, is_verified
		)
		VALUES ($1, $2, $3, $4, COALESCE($5, 5.0), COALESCE($6, 0), COALESCE($7, 0),
				COALESCE($8, 'OFFLINE'), COALESCE($9, false))
		RETURNING id, created_at, updated_at, license_number, vehicle_type,
		          vehicle_attrs, rating, total_rides, total_earnings, status, is_verified
	`

	err := tx.QueryRow(
		ctx,
		query,
		driver.ID,
		driver.LicenseNumber,
		driver.VehicleType,
		vehicleAttrsJSON,
		driver.Rating,
		driver.TotalRides,
		driver.TotalEarnings,
		driver.Status,
		driver.IsVerified,
	).Scan(
		&created.ID,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.LicenseNumber,
		&created.VehicleType,
		&vehicleAttrsJSON,
		&created.Rating,
		&created.TotalRides,
		&created.TotalEarnings,
		&created.Status,
		&created.IsVerified,
	)
	if err != nil {
		return model.Driver{}, fmt.Errorf("failed to insert driver: %w", err)
	}

	if len(vehicleAttrsJSON) > 0 {
		if err := json.Unmarshal(vehicleAttrsJSON, &created.VehicleAttrs); err != nil {
			return model.Driver{}, fmt.Errorf("failed to unmarshal vehicle_attrs: %w", err)
		}
	}

	return created, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User

	query := `
		SELECT 
			id,
			created_at,
			updated_at,
			email,
			role,
			status,
			password_hash,
			attrs
		FROM users
		WHERE email = $1
	`

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.PasswordHash,
		&user.Attrs,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return model.User{}, fmt.Errorf("user not found: %w", err)
		}
		return model.User{}, fmt.Errorf("failed to fetch user by email: %w", err)
	}

	return user, nil
}
