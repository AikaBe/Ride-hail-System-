package registration

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/model"

	"github.com/jackc/pgx/v5"
)

func RegisterHandler(db *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" || req.Role == "" {
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
			http.Error(w, "failed to insert user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if req.Role == "DRIVER" {
			_, err = db.Exec(ctx, `
				INSERT INTO drivers (id, license_number, vehicle_type, vehicle_attrs, status)
				VALUES ($1, $2, $3, $4, 'OFFLINE')
			`, userID, req.LicenseNumber, req.VehicleType, req.VehicleAttrs)
			if err != nil {
				http.Error(w, "failed to insert driver: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		token, err := auth.GenerateToken(userID, req.Role)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.RegisterResponse{
			UserID: userID,
			Token:  token,
		})
	}
}
