package ride_service

import (
	"log"
	"net/http"
	"ride-hail/internal/common/db"
	ridehttp "ride-hail/internal/ride/http"
	"ride-hail/internal/ride/repository"
	"ride-hail/internal/ride/service"
)

func main() {
	pg, err := db.NewPostgres("localhost", 5432, "postgres", "password", "rides_db")
	if err != nil {
		log.Fatal(err)
	}
	defer pg.Close()

	repo := repository.NewRideRepository(pg)
	manager := service.NewRideManager(repo)
	handler := ridehttp.NewRideHandler(manager)

	mux := http.NewServeMux()
	ridehttp.SetupRoutes(mux, handler)

	log.Println("🚀 Server running on port 8080")
	http.ListenAndServe(":8080", mux)
}
