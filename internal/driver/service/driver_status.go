package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	commonmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/handler/dto"
	"ride-hail/internal/driver/model"
	"ride-hail/internal/driver/rmq"
	model2 "ride-hail/internal/ride/model"
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
	"time"
)

type DriverRepository interface {
	FindNearbyDrivers(ctx context.Context, pickup model.Location, vehicleType usermodel.VehicleType, radiusMeters float64) ([]model.DriverNearby, error)
	SetOnline(ctx context.Context, driverID uuid.UUID, lat, lon float64) (model.DriverSession, error)
	SetOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, error)
	SaveLocation(ctx context.Context, location model.LocationHistory) (model2.Coordinate, error)
	Start(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, loc model.Location) (usermodel.DriverStatus, time.Time, error)
	Complete(ctx context.Context, driverID uuid.UUID, driverEarning float64, location model.Location, distance, duration float64) (time.Time, error)
	GetRideStatus(ctx context.Context, driverID, rideID uuid.UUID) (model2.RideStatus, error)
	GetDriverStatus(ctx context.Context, driverID uuid.UUID) (usermodel.DriverStatus, error)
	GetInfo(ctx context.Context, id string) (model.DriverInfo, error)
	GetPickupLocation(ctx context.Context, rideID string) (float64, float64, error)
	GetDriverIDByRideID(ctx context.Context, rideID string) (string, error)
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

func (s *DriverService) ListenForRides(ctx context.Context, queueName string) {
	err := s.rmqClient.ConsumeRideRequests(queueName, func(msg commonmq.RideRequestedMessage) {
		log.Printf("🚕 Ride request received: %+v", msg)

		// Вместо поиска рядом — просто всем подключённым водителям
		s.wsHub.BroadcastRideOffer(msg)
	})
	if err != nil {
		log.Fatalf("Failed to start consuming ride requests: %v", err)
	}
}

func (s *DriverService) ListenForPassengers(ctx context.Context, queueName string) {
	err := s.rmqClient.ConsumePassengerInfo(queueName, func(msg commonmq.PassiNFO) {
		log.Printf("📨 Получен ответ от пасажира по заказу %s ", msg.RideID)

		data, _ := json.Marshal(msg)
		DriverID, err := s.repo.GetDriverIDByRideID(ctx, msg.RideID)
		if err != nil {
			log.Println(err)
		}
		driverId := "driver_" + DriverID
		log.Printf("Отправка водителю %s: %s", DriverID, string(data))
		s.wsHub.SendToClient(driverId, data)
	})

	if err != nil {
		log.Printf("Ошибка при чтении сообщений очереди %s: %v", queueName, err)
	}
}

func (s *DriverService) SendToMq(ctx context.Context) {
	log.Println("🛰️ Listening for driver responses from WebSocket...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopped listening for driver responses.")
			return

		case resp := <-s.wsHub.DriverResponses:
			log.Printf(" Received driver response from WS: %+v", resp)
			driverInfo, err := s.repo.GetInfo(ctx, resp.DriverID)
			if err != nil {

			}
			pickupLat, pickupLng, err := s.repo.GetPickupLocation(ctx, resp.RideID)
			if err != nil {
				log.Printf("❌ Не удалось получить координаты подачи: %v", err)
				continue
			}

			estimatedMinutes, estimatedArrival := estimateArrival(
				resp.CurrentLocation.Latitude,
				resp.CurrentLocation.Longitude,
				pickupLat,
				pickupLng,
			)
			msg := commonmq.DriverResponseMessage{
				RideID:                  resp.RideID,
				OfferID:                 resp.OfferID,
				DriverID:                resp.DriverID,
				Accepted:                resp.Accepted,
				EstimatedArrivalMinutes: estimatedMinutes,
				DriverLocation: commonmq.LatLng{
					Lat: resp.CurrentLocation.Latitude,
					Lng: resp.CurrentLocation.Longitude,
				},
				DriverInfo: commonmq.DriverInfo{
					Name:   driverInfo.Name,
					Rating: driverInfo.Rating,
					Vehicle: commonmq.Vehicle{
						Make:  driverInfo.Vehicle.Make,
						Model: driverInfo.Vehicle.Model,
						Color: driverInfo.Vehicle.Color,
						Plate: driverInfo.Vehicle.Plate,
					},
				},
				EstimatedArrival: estimatedArrival,
				RespondedAt:      time.Now(),
			}

			// Отправляем в MQ
			err = s.rmqClient.PublishDriverResponse(ctx, msg)
			if err != nil {
				log.Printf("❗ Failed to send driver response to MQ: %v", err)
			} else {
				log.Printf("✅ Sent driver response to MQ: %+v", msg)
			}
		}
	}
}

func (s *DriverService) UpdateLocationWS(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopped listening for driver responses.")
			return

		case resp := <-s.wsHub.UpdateLocation:
			log.Printf(" Received driver response from WS: %+v", resp)

			// Отправляем в MQ
			err := s.rmqClient.PublishLocationUpdate(ctx, resp)
			if err != nil {
				log.Printf("❗ Failed to update location: %v", err)
			} else {
				log.Printf("✅ Sent driver update location: %+v", resp)
			}
		}
	}
}

func calculateDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // радиус Земли в км
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
func estimateArrival(driverLat, driverLng, pickupLat, pickupLng float64) (int, time.Time) {
	distanceKm := calculateDistanceKm(driverLat, driverLng, pickupLat, pickupLng)
	speedKmH := 40.0

	hours := distanceKm / speedKmH
	minutes := int(hours * 60)

	estimatedArrival := time.Now().Add(time.Duration(minutes) * time.Minute)
	return minutes, estimatedArrival
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
	if driverStatus != usermodel.DriverStatusOffline {
		return model.DriverSession{}, errors.New("driver is not offline")
	}
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, float64, error) {
	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		return model.DriverSession{}, 0, err
	}
	if driverStatus == usermodel.DriverStatusEnRoute || driverStatus == usermodel.DriverStatusBusy {
		return model.DriverSession{}, 0, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}
	session, err := s.repo.SetOffline(ctx, driverID)
	if err != nil {
		return model.DriverSession{}, 0, err
	}
	durationHours := time.Since(session.StartedAt).Hours()
	return session, durationHours, nil
}

func (s *DriverService) UpdateLocation(ctx context.Context, location model.LocationHistory) (model2.Coordinate, error) {
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

	coord, err := s.repo.SaveLocation(ctx, location)
	if err != nil {
		return model2.Coordinate{}, err
	}

	msg := commonmq.LocationUpdateMessage{
		DriverID: string(coord.EntityID),
		//RideID:    rideID,
		Location:  commonmq.LatLng{Lat: coord.Latitude, Lng: coord.Longitude},
		SpeedKmh:  location.SpeedKmh,
		Heading:   location.HeadingDegrees,
		Timestamp: coord.UpdatedAt.UTC(),
	}
	if err := s.rmqClient.PublishLocationUpdate(ctx, msg); err != nil {
		log.Printf("⚠️ Failed to publish driver location: %v", err)
	}
	return coord, err
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
	if driverStatus != usermodel.DriverStatusBusy {
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
		Status:        usermodel.DriverStatusAvailable,
		CompletedAt:   completedAt.Format(time.RFC3339),
		DriverEarning: driverEarnings,
		Message:       "Ride completed successfully",
	}
	return resp, nil
}

func (s *DriverService) GetDriverInfo(ctx context.Context, driverID string) (model.DriverInfo, error) {
	responce, err := s.repo.GetInfo(ctx, driverID)
	if err != nil {
		return model.DriverInfo{}, err
	}
	return responce, nil
}
