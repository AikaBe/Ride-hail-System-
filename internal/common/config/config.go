package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
	}
	RabbitMQ struct {
		Host     string
		Port     int
		User     string
		Password string
	}
	WebSocket struct {
		Port int
	}
	Services struct {
		RideServicePort           int
		DriverLocationServicePort int
		AdminServicePort          int
	}
}

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return def
}

func getEnvInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvInt("DB_PORT", 5432)
	cfg.Database.User = getEnv("DB_USER", "ridehail_user")
	cfg.Database.Password = getEnv("DB_PASSWORD", "ridehail_pass")
	cfg.Database.Name = getEnv("DB_NAME", "ridehail_db")

	cfg.RabbitMQ.Host = getEnv("RABBITMQ_HOST", "localhost")
	cfg.RabbitMQ.Port = getEnvInt("RABBITMQ_PORT", 5672)
	cfg.RabbitMQ.User = getEnv("RABBITMQ_USER", "guest")
	cfg.RabbitMQ.Password = getEnv("RABBITMQ_PASSWORD", "guest")

	cfg.WebSocket.Port = getEnvInt("WS_PORT", 8080)

	cfg.Services.RideServicePort = getEnvInt("RIDE_SERVICE_PORT", 3000)
	cfg.Services.DriverLocationServicePort = getEnvInt("DRIVER_LOCATION_SERVICE_PORT", 3001)
	cfg.Services.AdminServicePort = getEnvInt("ADMIN_SERVICE_PORT", 3004)

	return cfg, nil
}

func (c *Config) Print() {
	fmt.Printf("📦 Database: %s@%s:%d/%s\n", c.Database.User, c.Database.Host, c.Database.Port, c.Database.Name)
	fmt.Printf("🐇 RabbitMQ: amqp://%s:%s@%s:%d\n", c.RabbitMQ.User, c.RabbitMQ.Password, c.RabbitMQ.Host, c.RabbitMQ.Port)
	fmt.Printf("🌐 WebSocket Port: %d\n", c.WebSocket.Port)
	fmt.Printf("🧩 Services → ride:%d | driver:%d | admin:%d\n",
		c.Services.RideServicePort, c.Services.DriverLocationServicePort, c.Services.AdminServicePort)
}
