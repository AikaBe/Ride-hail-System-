package main

import (
	"log"
	"ride-hail/internal/common/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	cfg.Print()

	log.Printf("✅ Ride Service running on port %d\n", cfg.Services.RideServicePort)
}
