package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"ride-hail/internal/common/model"
	commonmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/repository"
	"ride-hail/internal/driver/rmq"
	"time"
)

type DriverRepository interface {
	SetOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error)
	SetOffline(ctx context.Context, driverID string) (model.OfflineResponse, error)
	Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error)
	Start(ctx context.Context, driverID string, rideID string, req model.Location) (model.StartResponse, error)
	Complete(ctx context.Context, driverID string, driverEarning float64, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error)
	GetRideStatus(ctx context.Context, driverID, rideID string) (string, error)
	GetDriverStatus(ctx context.Context, driverID string) (string, error)
}

type DriverService struct {
	repo      *repository.DriverRepository
	rmqClient *rmq.Client
	wsHub     *websocket.Hub
}

// NewDriverService создаёт новый экземпляр DriverService.
func NewDriverService(repo *repository.DriverRepository, rmqClient *rmq.Client, hub *websocket.Hub) *DriverService {
	return &DriverService{
		repo:      repo,
		rmqClient: rmqClient,
		wsHub:     hub,
	}
}

// RideEvent представляет данные о заказе.
type RideEvent struct {
	RideID      string `json:"ride_id"`
	PassengerID string `json:"passenger_id"`
	Pickup      string `json:"pickup"`
	Dropoff     string `json:"dropoff"`
}

// ListenForRides слушает очередь ride.requests и рассылает предложения водителям.
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

// sendRideOffers рассылает предложение всем найденным водителям
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

func (s *DriverService) GoOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error) {
	if lat < -90 || lat > 90 {
		return model.OnlineResponse{}, errors.New("latitude out of range")
	}
	if lon < -180 || lon > 180 {
		return model.OnlineResponse{}, errors.New("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.OnlineResponse{}, err
	}
	if driverStatus != "OFFLINE" {
		return model.OnlineResponse{}, errors.New("driver is not offline")
	}
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.OfflineResponse{}, err
	}
	if driverStatus == "EN_ROUTE" || driverStatus == "BUSY" {
		return model.OfflineResponse{}, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}
	return s.repo.SetOffline(ctx, driverID)
}

func (s *DriverService) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	if req.Latitude < -90 || req.Latitude > 90 {
		return model.LocationResponse{}, errors.New("latitude out of range")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return model.LocationResponse{}, errors.New("longitude out of range")
	}
	if req.AccuracyMeters > 50 || req.AccuracyMeters < 0 {
		return model.LocationResponse{}, errors.New("location accuracy too low or less than 0")
	}
	if req.SpeedKmh < 0 || req.SpeedKmh > 490 {
		return model.LocationResponse{}, errors.New("invalid speed ")
	}
	if req.HeadingDegrees < 0 || req.HeadingDegrees > 360 {
		return model.LocationResponse{}, errors.New("invalid heading")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.LocationResponse{}, err
	}
	if driverStatus == "OFFLINE" {
		return model.LocationResponse{}, errors.New("driver is OFFLINE")
	}
	return s.repo.Location(ctx, driverID, req)
}

func (s *DriverService) Start(ctx context.Context, driverID string, rideId string, location model.Location) (model.StartResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, rideId)
	if err != nil {
		return model.StartResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		return model.StartResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	if location.Latitude < -90 || location.Latitude > 90 {
		return model.StartResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return model.StartResponse{}, fmt.Errorf("longitude out of range")
	}
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.StartResponse{}, err
	}
	if driverStatus != "AVAILABLE" {
		return model.StartResponse{}, errors.New("driver is not available")
	}
	return s.repo.Start(ctx, driverID, rideId, location)
}

func (s *DriverService) Complete(ctx context.Context, driverID string, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error) {
	status, err := s.repo.GetRideStatus(ctx, driverID, req.RideID)
	if err != nil {
		return model.CompleteResponse{}, err
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		return model.CompleteResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}
	const baseFare = 400.0
	const perKmRate = 120.0
	const perMinuteRate = 20.0

	driverEarnings := baseFare +
		req.ActualDistanceKm*perKmRate +
		req.ActualDurationMins*perMinuteRate

	if location.Latitude < -90 || location.Latitude > 90 {
		return model.CompleteResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return model.CompleteResponse{}, fmt.Errorf("longitude out of range")
	}
	if req.ActualDurationMins <= 0 {
		return model.CompleteResponse{}, fmt.Errorf("duration out of range")
	}
	if req.ActualDistanceKm <= 0 {
		return model.CompleteResponse{}, fmt.Errorf("duration out of range")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.CompleteResponse{}, err
	}
	if driverStatus != "BUSY" {
		return model.CompleteResponse{}, errors.New("driver status not busy")
	}
	resp, err := s.repo.Complete(ctx, driverID, driverEarnings, req, location)
	if err != nil {
		return model.CompleteResponse{}, err
	}

	resp.DriverEarning = driverEarnings
	resp.Message = fmt.Sprintf("Ride completed successfully at %s", time.Now().Format(time.RFC3339))

	return resp, nil
}
