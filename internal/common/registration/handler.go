package registration

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/model"

	"github.com/jackc/pgx/v5"
)

func RegisterHandler(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")

		if r.Method != http.MethodPost {
			logger.Info("registration_invalid_method", "Only POST allowed", requestID, "")
			http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("registration_decode_failed", "Failed to decode request body", requestID, "", err.Error(), "")
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" || req.Role == "" {
			logger.Info("registration_missing_fields", "email, password or role missing", requestID, "")
			http.Error(w, "email, password and role are required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()

		var userID string
		err := db.QueryRow(ctx, `
	INSERT INTO users (email, role, password_hash, attrs)
	VALUES ($1, $2, $3, jsonb_build_object('name', CAST($4 AS text)))
	RETURNING id
`, req.Email, req.Role, req.Password, req.Name).Scan(&userID)
		if err != nil {
			logger.Error("registration_insert_user_failed", "Failed to insert user", requestID, req.Email, err.Error(), "")

			http.Error(w, "failed to insert user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if req.Role == "DRIVER" {
			_, err = db.Exec(ctx, `
				INSERT INTO drivers (id, license_number, vehicle_type, vehicle_attrs, status)
				VALUES ($1, $2, $3, $4, 'OFFLINE')
			`, userID, req.LicenseNumber, req.VehicleType, req.VehicleAttrs)
			if err != nil {
				logger.Error("registration_insert_driver_failed", "Failed to insert driver", requestID, userID, err.Error(), "")

				http.Error(w, "failed to insert driver: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		token, err := auth.GenerateToken(userID, req.Role)
		if err != nil {
			logger.Error("registration_token_failed", "Failed to generate token", requestID, userID, err.Error(), "")

			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}
		logger.Info("registration_success", "User registered successfully", requestID, userID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.RegisterResponse{
			UserID: userID,
			Token:  token,
		})
	}
}
