package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

func SetupRoutes(mux *http.ServeMux, rideHandler *RideHandler) {
	mux.HandleFunc("/rides", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			rideHandler.CreateRide(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/rides/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel") {
			rideID := extractRideIDFromPath(r.URL.Path)
			if rideID == "" {
				http.Error(w, "ride_id is required", http.StatusBadRequest)
				return
			}

			var req struct {
				Reason string `json:"reason"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			resp, err := rideHandler.RideService.CancelRide(r.Context(), rideID, req.Reason)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func extractRideIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
