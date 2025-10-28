package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	common "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/ride/model"
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"

	"ride-hail/internal/ride/repository"
	rmqClient "ride-hail/internal/ride/rmq"
	"time"

	"github.com/jackc/pgx/v5"
)

type RideRepository interface {
	InsertRide(ctx context.Context, tx pgx.Tx, ride model.Ride) (*model.Ride, error)
	InsertRideEvent(ctx context.Context, tx pgx.Tx, event model.RideEvent) error
	InsertCoordinate(ctx context.Context, tx pgx.Tx, coordinate model.Coordinate) (string, error)
	CancelRide(ctx context.Context, rideID, reason string) (*repository.CancelRideResponse, error)
	GetPassengerIDByRideID(ctx context.Context, rideID string) (string, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
	UpdateRideStatusMatched(ctx context.Context, rideID string, driverID string) error
}

type RideService struct {
	repo  RideRepository
	mq    *rmqClient.Client
	wsHub *websocket.Hub
}

func NewRideManager(repo RideRepository, mq *rmqClient.Client, wsHub *websocket.Hub) *RideService {
	return &RideService{repo: repo, mq: mq, wsHub: wsHub}
}

func (s *RideService) ListenForDriver(ctx context.Context, queueName string) {
	err := s.mq.ConsumeDriverResponses(queueName, func(msg common.DriverResponseMessage) {
		log.Printf("📨 Получен ответ от водителя %s по заказу %s (accepted=%v)",
			msg.DriverID, msg.RideID, msg.Accepted)

		// 🟢 Если водитель принял заказ
		if msg.Accepted {
			data, _ := json.Marshal(msg)

			passengerID, err := s.repo.GetPassengerIDByRideID(ctx, msg.RideID)
			if err != nil {
				log.Println(err)
			}
			passId := "passenger_" + passengerID
			log.Printf("Отправка пассажиру %s: %s", passengerID, string(data))
			s.wsHub.SendToClient(passId, data)
			err = s.repo.UpdateRideStatusMatched(ctx, msg.RideID, msg.DriverID)
			if err != nil {
				log.Println(err)
			}
			//go s.SendPassInfo(ctx)
			//if err != nil {
			//	log.Println(err)
			//}
		} else {
			// 🟥 Если водитель отклонил
			log.Printf("🚫 Водитель %s отклонил поездку %s", msg.DriverID, msg.RideID)
		}
	})

	if err != nil {
		log.Printf("Ошибка при чтении сообщений очереди %s: %v", queueName, err)
	}
}

func (s *RideService) SendPassInfo(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopped listening for passenger responses.")
			return

		case resp := <-s.wsHub.PassengerResponses:
			log.Printf(" Received passenger response from WS: %+v", resp)

			// Отправляем в MQ
			err := s.mq.PublishPassengerInfo(ctx, resp)
			if err != nil {
				log.Printf("Failed to send passenger response to MQ: %v", err)
			} else {
				log.Printf("Sent passenger response to MQ: %+v", resp)
			}
		}
	}
}

func (s *RideService) LocationUpdate(ctx context.Context, queueName string) {
	err := s.mq.ConsumeLocationUpdates(queueName, func(msg common.LocationUpdateMessage) {
		log.Printf("📨 геолокация изменилась %s по заказу %s ",
			msg.DriverID, msg.RideID)

		data, _ := json.Marshal(msg)

		passengerID, err := s.repo.GetPassengerIDByRideID(ctx, msg.RideID)
		if err != nil {
			log.Println(err)
		}
		passId := "passenger_" + passengerID
		log.Printf("Отправка пассажиру %s: %s", passengerID, string(data))
		s.wsHub.SendToClient(passId, data)
	})

	if err != nil {
		log.Printf("Ошибка при чтении сообщений очереди %s: %v", queueName, err)
	}

}
func (s *RideService) CreateRide(ctx context.Context, ride model.Ride, pickup, destination model.Coordinate) (*model.Ride, float64, int, error) {
	if err := s.validateRideRequest(ride); err != nil {
		return nil, 0, 0, err
	}
	if err := s.validateCoordinates(pickup, destination); err != nil {
		return nil, 0, 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	distanceKm, durationMin, err := calculateRoute(
		pickup.Latitude, pickup.Longitude,
		destination.Latitude, destination.Longitude,
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to calculate route: %w", err)
	}

	estimatedFare, err := calculateFare(*ride.VehicleType, distanceKm, float64(durationMin))
	if err != nil {
		return nil, 0, 0, err
	}

	rideNumber := fmt.Sprintf("RIDE_%s", time.Now().Format("20060102_150405"))

	pickup.EntityType = usermodel.EntityTypePassenger
	pickup.FareAmount = &estimatedFare
	pickup.DistanceKm = &distanceKm
	pickup.DurationMinute = &durationMin
	pickup.IsCurrent = false

	pickupCoordID, err := s.repo.InsertCoordinate(ctx, tx, pickup)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create pickup coordinate: %w", err)
	}

	destination.EntityType = usermodel.EntityTypePassenger
	destination.FareAmount = &estimatedFare
	destination.DistanceKm = &distanceKm
	destination.DurationMinute = &durationMin
	destination.IsCurrent = false

	destCoordID, err := s.repo.InsertCoordinate(ctx, tx, destination)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create destination coordinate: %w", err)
	}

	status := model.RideRequested
	pickupID := uuid.UUID(pickupCoordID)
	destID := uuid.UUID(destCoordID)

	ride.RideNumber = rideNumber
	ride.Status = &status
	ride.Priority = 1
	ride.EstimatedFare = &estimatedFare
	ride.PickupCoordinateID = &pickupID
	ride.DestinationCoordinateID = &destID

	createdRide, err := s.repo.InsertRide(ctx, tx, ride)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create ride: %w", err)
	}

	event := model.RideEvent{
		RideID:    string(createdRide.ID),
		EventType: model.EventRideRequested,
		EventData: json.RawMessage(fmt.Sprintf(`{
		"old_status": null,
		"new_status": "REQUESTED",
		"vehicle_type": "%s",
		"estimated_fare": %.2f,
		"pickup": {"lat": %.6f, "lng": %.6f},
		"destination": {"lat": %.6f, "lng": %.6f},
		"timestamp": "%s"
	}`,
			*ride.VehicleType,
			*ride.EstimatedFare,
			pickup.Latitude,
			pickup.Longitude,
			destination.Latitude,
			destination.Longitude,
			time.Now().UTC().Format(time.RFC3339),
		)),
	}
	if err := s.repo.InsertRideEvent(ctx, tx, event); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create ride event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	message := common.RideRequestedMessage{
		RideID:     string(createdRide.ID),
		RideNumber: rideNumber,
		PickupLocation: common.Location{
			Lat:     pickup.Latitude,
			Lng:     pickup.Longitude,
			Address: pickup.Address,
		},
		DestinationLocation: common.Location{
			Lat:     destination.Latitude,
			Lng:     destination.Longitude,
			Address: destination.Address,
		},
		RideType:       *createdRide.VehicleType,
		MaxDistanceKm:  distanceKm,
		TimeoutSeconds: 30,
		CorrelationID:  string(createdRide.ID),
	}

	if err := s.mq.PublishRideRequested(ctx, message); err != nil {
		fmt.Printf("WARN: failed to publish ride.request event: %v\n", err)
	}

	return createdRide, distanceKm, durationMin, nil
}

func (s *RideService) CancelRide(ctx context.Context, rideID, reason string) (*repository.CancelRideResponse, error) {
	if rideID == "" {
		return nil, fmt.Errorf("ride_id is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("cancellation reason is required")
	}

	return s.repo.CancelRide(ctx, rideID, reason)
}

func validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func calculateRoute(pickupLat, pickupLng, destLat, destLng float64) (distanceKm float64, durationMin int, err error) {
	const earthRadiusKm = 6371.0

	if pickupLat < -90 || pickupLat > 90 || destLat < -90 || destLat > 90 ||
		pickupLng < -180 || pickupLng > 180 || destLng < -180 || destLng > 180 {
		return 0, 0, errors.New("invalid latitude or longitude range")
	}

	lat1 := degreesToRadians(pickupLat)
	lng1 := degreesToRadians(pickupLng)
	lat2 := degreesToRadians(destLat)
	lng2 := degreesToRadians(destLng)

	dlat := lat2 - lat1
	dlng := lng2 - lng1
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceKm = earthRadiusKm * c

	// Средняя скорость: 30 км/ч
	duration := (distanceKm / 30.0) * 60.0
	durationMin = int(math.Ceil(duration))

	if durationMin < 1 {
		durationMin = 1
	}

	return distanceKm, durationMin, nil
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func calculateFare(rideType usermodel.VehicleType, distanceKm, durationMin float64) (float64, error) {
	var baseFare, perKm, perMin float64

	switch rideType {
	case usermodel.VehicleEconomy:
		baseFare, perKm, perMin = 500, 100, 50
	case usermodel.VehiclePremium:
		baseFare, perKm, perMin = 800, 120, 60
	case usermodel.VehicleXL:
		baseFare, perKm, perMin = 1000, 150, 75
	default:
		return 0, fmt.Errorf("invalid ride_type: %s", rideType)
	}

	return baseFare + (distanceKm * perKm) + (durationMin * perMin), nil
}
