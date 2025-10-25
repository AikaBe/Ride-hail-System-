package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	commonmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/uuid"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/handler/dto"
	"ride-hail/internal/driver/model"
	"ride-hail/internal/driver/rmq"
	model2 "ride-hail/internal/ride/model"
	"time"
)

type DriverRepository interface {
	FindNearbyDrivers(ctx context.Context, pickup model.Location, vehicleType model2.VehicleType, radiusMeters float64) ([]model.DriverNearby, error)
	SetOnline(ctx context.Context, driverID uuid.UUID, lat, lon float64) (model.DriverSession, error)
	SetOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, error)
	Location(ctx context.Context, location model.LocationHistory) (model2.Coordinate, error)
	Start(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, loc model.Location) (model.DriverStatus, time.Time, error)
	Complete(ctx context.Context, driverID uuid.UUID, driverEarning float64, location model.Location, distance, duration float64) (time.Time, error)
	GetRideStatus(ctx context.Context, driverID, rideID uuid.UUID) (model2.RideStatus, error)
	GetDriverStatus(ctx context.Context, driverID uuid.UUID) (model.DriverStatus, error)
}

type DriverService struct {
	repo      DriverRepository
	rmqClient *rmq.Client
	wsHub     *websocket.Hub
}

func NewDriverService(repo DriverRepository, rmqClient *rmq.Client, hub *websocket.Hub) *DriverService {
	return &DriverService{
		repo:      repo,
		rmqClient: rmqClient,
		wsHub:     hub,
	}
}

type RideEvent struct {
	RideID      string `json:"ride_id"`
	PassengerID string `json:"passenger_id"`
	Pickup      string `json:"pickup"`
	Dropoff     string `json:"dropoff"`
}

func (s *DriverService) ListenForRides(ctx context.Context, queueName string) {
	err := s.rmqClient.ConsumeRideRequests(queueName, func(msg commonmq.RideRequestedMessage) {
		log.Printf("🚕 Получен новый запрос поездки: %+v", msg)

		pickup := model.Location{Latitude: msg.PickupLocation.Lat, Longitude: msg.PickupLocation.Lng}
		vehicleType := msg.RideType

		radius := 5_000.0
		for {
			drivers, err := s.repo.FindNearbyDrivers(ctx, pickup, vehicleType, radius)
			if err != nil {
				log.Printf(" Ошибка при поиске водителей: %v", err)
				return
			}

			if len(drivers) > 0 {
				log.Printf("Найдено %d водителей в радиусе %.0f м", len(drivers), radius)
				s.sendRideOffers(drivers, msg)
				return
			}

			if radius >= 15_000 {
				log.Printf("Водителей не найдено даже в радиусе %.0f м", radius)
				return
			}

			radius += 1_000
			log.Printf("Увеличиваем радиус до %.0f м и пробуем снова...", radius)
			time.Sleep(2 * time.Second)
		}
	})

	if err != nil {
		log.Fatalf("Ошибка при запуске ConsumeRideRequests: %v", err)
	}
}

func (s *DriverService) sendRideOffers(drivers []model.DriverNearby, msg commonmq.RideRequestedMessage) {
	for _, d := range drivers {
		data, _ := json.Marshal(msg)
		s.wsHub.SendToClient(d.ID, data)
		log.Printf(" Ride offer отправлено водителю %s (%.3f км)", d.ID, d.Distance)
	}
	timeout := time.After(30 * time.Second) // ⏰ Ограничение ожидания

	for {
		select {
		case resp := <-s.wsHub.DriverResponses:
			// Проверяем, что ответ относится к текущему заказу
			if resp.RideID != msg.RideID {
				continue
			}

			if resp.Accepted {
				log.Printf("✅ Водитель %s принял заказ %s", resp.DriverID, resp.RideID)

				// Уведомляем остальных водителей, что заказ занят
				for _, d := range drivers {
					if d.ID != resp.DriverID {
						busyMsg := map[string]interface{}{
							"type":    "ride_unavailable",
							"ride_id": msg.RideID,
						}
						data, _ := json.Marshal(busyMsg)
						s.wsHub.SendToClient(d.ID, data)
					}
				}

				// Публикуем ответ водителя в брокер
				_, err := s.HandleDriverResponse(
					context.Background(),
					resp.DriverID,
					resp.RideID,
					"", // offerID можно добавить позже
					true,
					resp.EstimatedArrivalMinutes,
					commonmq.LatLng{},     // можно передавать реальные координаты
					commonmq.DriverInfo{}, // можно дополнить
				)
				if err != nil {
					log.Printf("⚠️ Ошибка публикации ответа водителя: %v", err)
				}
				return

			} else {
				log.Printf("🚫 Водитель %s отклонил заказ %s", resp.DriverID, resp.RideID)
			}

		case <-timeout:
			log.Println("⏰ Время ожидания ответов от водителей истекло — никто не принял заказ")
			return
		}
	}
}

// HandleDriverResponse публикует ответ водителя (accept/decline) в брокер.
func (s *DriverService) HandleDriverResponse(
	ctx context.Context,
	driverID string,
	rideID string,
	offerID string,
	accepted bool,
	arrivalMinutes int,
	driverLocation commonmq.LatLng,
	driverInfo commonmq.DriverInfo,
) (commonmq.DriverResponseMessage, error) {
	resp := commonmq.DriverResponseMessage{
		RideID:                  rideID,
		OfferID:                 offerID,
		DriverID:                driverID,
		Accepted:                accepted,
		EstimatedArrivalMinutes: arrivalMinutes,
		DriverLocation:          driverLocation,
		DriverInfo:              driverInfo,
		EstimatedArrival:        time.Now().Add(time.Duration(arrivalMinutes) * time.Minute),
		RespondedAt:             time.Now(),
	}

	// Публикуем сообщение в RabbitMQ
	if err := s.rmqClient.PublishDriverResponse(ctx, resp); err != nil {
		return resp, fmt.Errorf("failed to publish driver response: %w", err)
	}

	status := "declined"
	if accepted {
		status = "accepted"
	}
	log.Printf("Водитель %s %s поездку %s (offer: %s, прибытие через %d мин)",
		driverID, status, rideID, offerID, arrivalMinutes)

	return resp, nil
}

func (s *DriverService) GoOnline(ctx context.Context, driverID uuid.UUID, lat, lon float64) (model.DriverSession, error) {
	if lat < -90 || lat > 90 {
		return model.DriverSession{}, errors.New("latitude out of range")
	}
	if lon < -180 || lon > 180 {
		return model.DriverSession{}, errors.New("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.DriverSession{}, err
	}
	if driverStatus != model.DriverStatusOffline {
		return model.DriverSession{}, errors.New("driver is not offline")
	}
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, float64, error) {
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.DriverSession{}, 0, err
	}
	if driverStatus == model.DriverStatusEnRoute || driverStatus == model.DriverStatusBusy {
		return model.DriverSession{}, 0, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}
	session, err := s.repo.SetOffline(ctx, driverID)
	if err != nil {
		return model.DriverSession{}, 0, err
	}
	durationHours := time.Since(session.StartedAt).Hours()
	return session, durationHours, nil
}

func (s *DriverService) Location(ctx context.Context, location model.LocationHistory) (model2.Coordinate, error) {
	if location.Latitude < -90 || location.Latitude > 90 {
		return model2.Coordinate{}, errors.New("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return model2.Coordinate{}, errors.New("longitude out of range")
	}
	if location.AccuracyMeters > 50 || location.AccuracyMeters < 0 {
		return model2.Coordinate{}, errors.New("location accuracy too low or less than 0")
	}
	if location.SpeedKmh < 0 || location.SpeedKmh > 490 {
		return model2.Coordinate{}, errors.New("invalid speed ")
	}
	if location.HeadingDegrees < 0 || location.HeadingDegrees > 360 {
		return model2.Coordinate{}, errors.New("invalid heading")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, uuid.UUID(location.DriverID))
	if err != nil {
		return model2.Coordinate{}, err
	}
	if driverStatus == "OFFLINE" {
		return model2.Coordinate{}, errors.New("driver is OFFLINE")
	}
	return s.repo.Location(ctx, location)
}

func (s *DriverService) Start(ctx context.Context, driverID uuid.UUID, rideId uuid.UUID, location model.Location) (dto.StartResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, rideId)
	if err != nil {
		return dto.StartResponse{}, err
	}

	if status == model2.RideCompleted || status == model2.RideCancelled {
		return dto.StartResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	if location.Latitude < -90 || location.Latitude > 90 {
		return dto.StartResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return dto.StartResponse{}, fmt.Errorf("longitude out of range")
	}

	currentDStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return dto.StartResponse{}, err
	}

	if currentDStatus != "AVAILABLE" {
		return dto.StartResponse{}, errors.New("driver is not available")
	}

	newDStatus, startedAt, err := s.repo.Start(ctx, driverID, rideId, location)
	resp := dto.StartResponse{
		RideID:    string(rideId),
		Status:    newDStatus,
		StartedAt: startedAt.Format(time.RFC3339),
		Message:   "Ride started successfully",
	}

	return resp, err
}

func (s *DriverService) Complete(ctx context.Context, driverID uuid.UUID, req dto.CompleteRequest) (dto.CompleteResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, req.RideID)
	if err != nil {
		return dto.CompleteResponse{}, err
	}
	if status == model2.RideCompleted || status == model2.RideCancelled {
		return dto.CompleteResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	const baseFare = 400.0
	const perKmRate = 120.0
	const perMinuteRate = 20.0

	driverEarnings := baseFare +
		req.ActualDistanceKm*perKmRate +
		req.ActualDurationMins*perMinuteRate

	if req.FinalLocation.Latitude < -90 || req.FinalLocation.Latitude > 90 {
		return dto.CompleteResponse{}, fmt.Errorf("latitude out of range")
	}
	if req.FinalLocation.Longitude < -180 || req.FinalLocation.Longitude > 180 {
		return dto.CompleteResponse{}, fmt.Errorf("longitude out of range")
	}
	if req.ActualDurationMins <= 0 {
		return dto.CompleteResponse{}, fmt.Errorf("duration out of range")
	}
	if req.ActualDistanceKm <= 0 {
		return dto.CompleteResponse{}, fmt.Errorf("duration out of range")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return dto.CompleteResponse{}, err
	}
	if driverStatus != model.DriverStatusBusy {
		return dto.CompleteResponse{}, errors.New("driver status not busy")
	}

	location := model.Location{
		Latitude:  req.FinalLocation.Latitude,
		Longitude: req.FinalLocation.Longitude,
	}
	completedAt, err := s.repo.Complete(ctx, driverID, driverEarnings, location, req.ActualDistanceKm, req.ActualDurationMins)
	if err != nil {
		return dto.CompleteResponse{}, err
	}
	resp := dto.CompleteResponse{
		RideID:        string(req.RideID),
		Status:        model.DriverStatusAvailable,
		CompletedAt:   completedAt.Format(time.RFC3339),
		DriverEarning: driverEarnings,
		Message:       "Ride completed successfully",
	}
	return resp, nil
}
