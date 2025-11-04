package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"ride-hail-system/internal/common/logger"
	"ride-hail-system/internal/common/websocket"
	"ride-hail-system/internal/ride/model"
	"ride-hail-system/internal/ride/repository"
	"ride-hail-system/pkg/uuid"

	common "ride-hail-system/internal/common/rmq"

	usermodel "ride-hail-system/internal/user/model"

	rmqClient "ride-hail-system/internal/ride/rmq"

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
	UpdateLocation(ctx context.Context, rideID, passengerID string) error
}

type RideService struct {
	repo  RideRepository
	mq    *rmqClient.Client
	wsHub *websocket.Hub
}

func NewRideManager(repo RideRepository, mq *rmqClient.Client, wsHub *websocket.Hub) *RideService {
	logger.SetServiceName("ride-service")
	return &RideService{repo: repo, mq: mq, wsHub: wsHub}
}

func (s *RideService) ListenForDriver(ctx context.Context, queueName string) {
	err := s.mq.ConsumeDriverResponses(queueName, func(msg common.DriverResponseMessage) {
		logger.Info("driver_response_received",
			fmt.Sprintf("получен ответ от водителя %s по заказу %s (accepted=%v)", msg.DriverID, msg.RideID, msg.Accepted),
			"", msg.RideID)

		if msg.Accepted {
			data, _ := json.Marshal(msg)

			passengerID, err := s.repo.GetPassengerIDByRideID(ctx, msg.RideID)
			if err != nil {
				logger.Error("get_passenger_id_failed", "не удалось получить passenger_id", "", msg.RideID, err.Error())
				return
			}

			err = s.repo.UpdateRideStatusMatched(ctx, msg.RideID, msg.DriverID)
			if err != nil {
				logger.Error("update_status_failed", "ошибка при обновлении статуса поездки", "", msg.RideID, err.Error())
			}

			passId := "passenger_" + passengerID
			logger.Info("send_to_passenger",
				fmt.Sprintf("отправка пассажиру %s: %s", passengerID, string(data)),
				"", msg.RideID)
			s.wsHub.SendToClient(passId, data)
		} else {
			logger.Warn("driver_declined", "водитель отклонил поездку", "", msg.RideID,
				fmt.Sprintf("driver_id=%s", msg.DriverID))
		}
	})
	if err != nil {
		logger.Error("consume_driver_responses_failed",
			fmt.Sprintf("ошибка при чтении сообщений очереди %s", queueName),
			"", "", err.Error())
	}
}

func (s *RideService) SendPassInfo(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("stop_listening", "остановлено получение ответов от пассажиров", "", "")
			return

		case resp := <-s.wsHub.PassengerResponses:
			logger.Debug("passenger_ws_response",
				fmt.Sprintf("получен ответ пассажира из WS: %+v", resp),
				"", resp.RideID)

			err := s.mq.PublishPassengerInfo(ctx, resp)
			if err != nil {
				logger.Error("mq_publish_failed", "ошибка отправки ответа пассажира в MQ", "", resp.RideID, err.Error())
			} else {
				logger.Info("mq_publish_success",
					fmt.Sprintf("успешно отправлен ответ пассажира в MQ: %+v", resp),
					"", resp.RideID)
			}
		}
	}
}

func (s *RideService) LocationUpdate(ctx context.Context, queueName string) {
	err := s.mq.ConsumeLocationUpdates(queueName, func(msg common.LocationUpdateMessage) {
		logger.Info("location_update_received",
			fmt.Sprintf("геолокация изменилась %s по заказу %s", msg.DriverID, msg.RideID),
			"", msg.RideID)

		data, _ := json.Marshal(msg)

		passengerID, err := s.repo.GetPassengerIDByRideID(ctx, msg.RideID)
		if err != nil {
			logger.Error("get_passenger_id_failed", "не удалось получить passenger_id", "", msg.RideID, err.Error())
			return
		}

		err = s.repo.UpdateRideStatusMatched(ctx, msg.RideID, msg.DriverID)
		if err != nil {
			logger.Error("insert updated location", "cannot insert new location to db", "", msg.RideID, err.Error())
			return
		}

		err = s.repo.UpdateLocation(ctx, msg.RideID, passengerID)
		if err != nil {
			logger.Error("insert updated status", "cannot update ride event", "", msg.RideID, err.Error())
			return
		}
		passId := "passenger_" + passengerID
		logger.Info("send_location_to_passenger",
			fmt.Sprintf("отправка пассажиру %s: %s", passengerID, string(data)),
			"", msg.RideID)

		s.wsHub.SendToClient(passId, data)
	})
	if err != nil {
		logger.Error("consume_location_failed",
			fmt.Sprintf("ошибка при чтении сообщений очереди %s", queueName),
			"", "", err.Error())
	}
}

func (s *RideService) CreateRide(ctx context.Context, ride model.Ride, pickup, destination model.Coordinate) (*model.Ride, float64, int, error) {
	logger.Info("create_ride_start", "начало создания поездки", "", "")

	if err := s.validateRideRequest(ride); err != nil {
		logger.Warn("invalid_ride_request", "некорректный запрос поездки", "", "", err.Error())
		return nil, 0, 0, err
	}
	if err := s.validateCoordinates(pickup, destination); err != nil {
		logger.Warn("invalid_coordinates", "некорректные координаты", "", "", err.Error())
		return nil, 0, 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		logger.Error("begin_tx_failed", "не удалось начать транзакцию", "", "", err.Error())
		return nil, 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			logger.Warn("tx_rollback", "откат транзакции из-за ошибки", "", "", err.Error())
		}
	}()

	distanceKm, durationMin, err := calculateRoute(
		pickup.Latitude, pickup.Longitude,
		destination.Latitude, destination.Longitude,
	)
	if err != nil {
		logger.Error("calculate_route_failed", "ошибка расчёта маршрута", "", "", err.Error())
		return nil, 0, 0, err
	}

	estimatedFare, err := calculateFare(*ride.VehicleType, distanceKm, float64(durationMin))
	if err != nil {
		logger.Error("calculate_fare_failed", "ошибка расчёта стоимости", "", "", err.Error())
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
		logger.Error("insert_pickup_failed", "ошибка вставки координаты точки отправления", "", "", err.Error())
		return nil, 0, 0, err
	}

	destination.EntityType = usermodel.EntityTypePassenger
	destination.FareAmount = &estimatedFare
	destination.DistanceKm = &distanceKm
	destination.DurationMinute = &durationMin
	destination.IsCurrent = false

	destCoordID, err := s.repo.InsertCoordinate(ctx, tx, destination)
	if err != nil {
		logger.Error("insert_destination_failed", "ошибка вставки координаты назначения", "", "", err.Error())
		return nil, 0, 0, err
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
		logger.Error("insert_ride_failed", "ошибка вставки поездки в БД", "", "", err.Error())
		return nil, 0, 0, err
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
		logger.Error("insert_ride_event_failed", "ошибка вставки события поездки", "", "", err.Error())
		return nil, 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("tx_commit_failed", "ошибка коммита транзакции", "", "", err.Error())
		return nil, 0, 0, err
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
		logger.Warn("publish_ride_request_failed", "не удалось опубликовать событие ride.request", "", string(createdRide.ID), err.Error())
	}

	logger.Info("create_ride_success", "поездка успешно создана и опубликована", "", string(createdRide.ID))
	return createdRide, distanceKm, durationMin, nil
}

func (s *RideService) CancelRide(ctx context.Context, rideID, reason string) (*repository.CancelRideResponse, error) {
	if rideID == "" {
		logger.Warn("CancelRide", "ride_id is required", "", rideID, "ride_id is empty")
		return nil, fmt.Errorf("ride_id is required")
	}
	if reason == "" {
		logger.Warn("CancelRide", "cancellation reason is required", "", rideID, "reason is empty")
		return nil, fmt.Errorf("cancellation reason is required")
	}

	logger.Info("CancelRide", fmt.Sprintf("Cancelling ride %s with reason: %s", rideID, reason), "", rideID)

	resp, err := s.repo.CancelRide(ctx, rideID, reason)
	if err != nil {
		logger.Error("CancelRide", "failed to cancel ride", "", rideID, err.Error())
		return nil, err
	}

	logger.Info("CancelRide", fmt.Sprintf("Ride %s successfully cancelled", rideID), "", rideID)
	return resp, nil
}

func calculateRoute(pickupLat, pickupLng, destLat, destLng float64) (distanceKm float64, durationMin int, err error) {
	const earthRadiusKm = 6371.0

	if pickupLat < -90 || pickupLat > 90 || destLat < -90 || destLat > 90 ||
		pickupLng < -180 || pickupLng > 180 || destLng < -180 || destLng > 180 {
		logger.Warn("calculateRoute", "invalid latitude or longitude range", "", "", "coordinates out of range")
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

	logger.Debug("calculateRoute", fmt.Sprintf("Route calculated: %.2f km, %d min", distanceKm, durationMin), "", "")
	return distanceKm, durationMin, nil
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

// 💰 Расчет стоимости поездки
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
		logger.Warn("calculateFare", "invalid ride_type", "", "", string(rideType))
		return 0, fmt.Errorf("invalid ride_type: %s", rideType)
	}

	fare := baseFare + (distanceKm * perKm) + (durationMin * perMin)
	logger.Debug("calculateFare", fmt.Sprintf("Fare calculated: %.2f (%s)", fare, rideType), "", "")
	return fare, nil
}
