package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ride-hail/internal/common/logger"
	commonmq "ride-hail/internal/common/rmq"
	"ride-hail/internal/common/websocket"
	"ride-hail/internal/driver/handler/dto"
	"ride-hail/internal/driver/model"
	"ride-hail/internal/driver/rmq"
	model2 "ride-hail/internal/ride/model"
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
)

type DriverRepository interface {
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
		logger.Info("listen_for_rides", "Ride request received", "", msg.RideID)
		s.wsHub.BroadcastRideOffer(msg)
	})
	if err != nil {
		logger.Error("listen_for_rides", "Failed to start consuming ride requests", "", "", err.Error())
	}
}

func (s *DriverService) ListenForPassengers(ctx context.Context, queueName string) {
	err := s.rmqClient.ConsumePassengerInfo(queueName, func(msg commonmq.PassiNFO) {
		logger.Info("listen_for_passengers", fmt.Sprintf("Passenger response received for ride %s", msg.RideID), "", msg.RideID)

		data, _ := json.Marshal(msg)
		driverID, err := s.repo.GetDriverIDByRideID(ctx, msg.RideID)
		if err != nil {
			logger.Error("listen_for_passengers", "Failed to get driver ID by ride ID", "", msg.RideID, err.Error())
			return
		}

		s.wsHub.SendToClient("driver_"+driverID, data)
		logger.Info("listen_for_passengers", fmt.Sprintf("Sent passenger info to driver %s", driverID), "", msg.RideID)
	})
	if err != nil {
		logger.Error("listen_for_passengers", fmt.Sprintf("Failed to consume queue %s", queueName), "", "", err.Error())
	}
}

func (s *DriverService) SendToMq(ctx context.Context) {
	logger.Info("send_to_mq", "Listening for driver responses from WebSocket...", "", "")

	for {
		select {
		case <-ctx.Done():
			logger.Warn("send_to_mq", "Stopped listening for driver responses", "", "", "")
			return

		case resp := <-s.wsHub.DriverResponses:
			logger.Debug("send_to_mq", fmt.Sprintf("Received driver response from WS: %+v", resp), "", resp.RideID)

			if strings.HasPrefix(resp.DriverID, "driver_") {
				resp.DriverID = strings.TrimPrefix(resp.DriverID, "driver_")
			}

			driverStatus, err := s.repo.GetDriverStatus(ctx, uuid.UUID(resp.DriverID))
			if err != nil {
				logger.Error("send_to_mq", "Failed to get driver status", "", resp.DriverID, err.Error())
				continue
			}
			if driverStatus != "AVAILABLE" {
				logger.Error("send_to_mq", "Driver doesn't available", resp.DriverID, resp.RideID, err.Error())
				continue
			}
			driverInfo, err := s.repo.GetInfo(ctx, resp.DriverID)
			if err != nil {
				logger.Error("send_to_mq", "Failed to get driver info", resp.DriverID, resp.RideID, err.Error())
				continue
			}

			pickupLat, pickupLng, err := s.repo.GetPickupLocation(ctx, resp.RideID)
			if err != nil {
				logger.Error("send_to_mq", "Failed to get pickup coordinates", resp.DriverID, resp.RideID, err.Error())
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
					Rating: driverInfo.Rating,
					Vehicle: commonmq.Vehicle{
						Year:  driverInfo.Vehicle.Year,
						Model: driverInfo.Vehicle.Model,
						Color: driverInfo.Vehicle.Color,
						Brand: driverInfo.Vehicle.Brand,
					},
				},
				EstimatedArrival: estimatedArrival,
				RespondedAt:      time.Now(),
			}

			if err := s.rmqClient.PublishDriverResponse(ctx, msg); err != nil {
				logger.Error("send_to_mq", "Failed to send driver response to MQ", resp.DriverID, resp.RideID, err.Error())
			} else {
				logger.Info("send_to_mq", "Sent driver response to MQ", resp.DriverID, resp.RideID)
			}
		}
	}
}

func (s *DriverService) UpdateLocationWS(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Warn("update_location_ws", "Stopped listening for driver location updates", "", "", "")
			return

		case resp := <-s.wsHub.UpdateLocation:
			logger.Debug("update_location_ws", fmt.Sprintf("Received driver location from WS: %+v", resp), resp.DriverID, "")

			if err := s.rmqClient.PublishLocationUpdate(ctx, resp); err != nil {
				logger.Error("update_location_ws", "Failed to publish driver location update", resp.DriverID, "", err.Error())
			} else {
				logger.Info("update_location_ws", "Published driver location update", resp.DriverID, "")
			}
		}
	}
}

func calculateDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
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
	return minutes, time.Now().Add(time.Duration(minutes) * time.Minute)
}

func (s *DriverService) GoOnline(ctx context.Context, driverID uuid.UUID, lat, lon float64) (model.DriverSession, error) {
	logger.Info("GoOnline", fmt.Sprintf("Driver %s requested to go online", driverID), "", "")

	if lat < -90 || lat > 90 {
		logger.Warn("GoOnline", "Invalid latitude", "", "", "latitude out of range")
		return model.DriverSession{}, errors.New("latitude out of range")
	}
	if lon < -180 || lon > 180 {
		logger.Warn("GoOnline", "Invalid longitude", "", "", "longitude out of range")
		return model.DriverSession{}, errors.New("longitude out of range")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error("GoOnline", "Failed to get driver status", "", "", err.Error())
		return model.DriverSession{}, err
	}

	if driverStatus != usermodel.DriverStatusOffline {
		logger.Warn("GoOnline", "Driver is not offline", "", "", fmt.Sprintf("status: %s", driverStatus))
		return model.DriverSession{}, errors.New("driver is not offline")
	}

	session, err := s.repo.SetOnline(ctx, driverID, lat, lon)
	if err != nil {
		logger.Error("GoOnline", "Failed to set driver online", "", "", err.Error())
		return model.DriverSession{}, err
	}

	logger.Info("GoOnline", fmt.Sprintf("Driver %s is now ONLINE", driverID), "", "")
	return session, nil
}

func (s *DriverService) GoOffline(ctx context.Context, driverID uuid.UUID) (model.DriverSession, float64, error) {
	logger.Info("GoOffline", fmt.Sprintf("Driver %s requested to go offline", driverID), "", "")

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error("GoOffline", "Failed to get driver status", "", "", err.Error())
		return model.DriverSession{}, 0, err
	}

	if driverStatus == usermodel.DriverStatusEnRoute || driverStatus == usermodel.DriverStatusBusy {
		logger.Warn("GoOffline", "Driver cannot go offline", "", "", fmt.Sprintf("status: %s", driverStatus))
		return model.DriverSession{}, 0, errors.New("driver cannot go offline(driver status: EN_ROUTE or BUSY)")
	}

	session, err := s.repo.SetOffline(ctx, driverID)
	if err != nil {
		logger.Error("GoOffline", "Failed to set driver offline", "", "", err.Error())
		return model.DriverSession{}, 0, err
	}

	durationHours := time.Since(session.StartedAt).Hours()
	logger.Info("GoOffline", fmt.Sprintf("Driver %s went offline after %.2f hours", driverID, durationHours), "", "")
	return session, durationHours, nil
}

func (s *DriverService) UpdateLocation(ctx context.Context, location model.LocationHistory) (model2.Coordinate, error) {
	logger.Debug("UpdateLocation", fmt.Sprintf("Updating location for driver %s", location.DriverID), "", "")

	if location.Latitude < -90 || location.Latitude > 90 {
		logger.Warn("UpdateLocation", "Invalid latitude", "", "", "latitude out of range")
		return model2.Coordinate{}, errors.New("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		logger.Warn("UpdateLocation", "Invalid longitude", "", "", "longitude out of range")
		return model2.Coordinate{}, errors.New("longitude out of range")
	}
	if location.AccuracyMeters > 50 || location.AccuracyMeters < 0 {
		logger.Warn("UpdateLocation", "Invalid accuracy", "", "", "location accuracy too low or less than 0")
		return model2.Coordinate{}, errors.New("location accuracy too low or less than 0")
	}
	if location.SpeedKmh < 0 || location.SpeedKmh > 490 {
		logger.Warn("UpdateLocation", "Invalid speed", "", "", "invalid speed")
		return model2.Coordinate{}, errors.New("invalid speed ")
	}
	if location.HeadingDegrees < 0 || location.HeadingDegrees > 360 {
		logger.Warn("UpdateLocation", "Invalid heading", "", "", "invalid heading")
		return model2.Coordinate{}, errors.New("invalid heading")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, uuid.UUID(location.DriverID))
	if err != nil {
		logger.Error("UpdateLocation", "Failed to get driver status", "", "", err.Error())
		return model2.Coordinate{}, err
	}
	if driverStatus == "OFFLINE" {
		logger.Warn("UpdateLocation", "Driver is offline", "", "", "driver is OFFLINE")
		return model2.Coordinate{}, errors.New("driver is OFFLINE")
	}

	coord, err := s.repo.SaveLocation(ctx, location)
	if err != nil {
		logger.Error("UpdateLocation", "Failed to save location", "", "", err.Error())
		return model2.Coordinate{}, err
	}

	msg := commonmq.LocationUpdateMessage{
		DriverID:  string(coord.EntityID),
		Location:  commonmq.LatLng{Lat: coord.Latitude, Lng: coord.Longitude},
		SpeedKmh:  location.SpeedKmh,
		Heading:   location.HeadingDegrees,
		Timestamp: coord.UpdatedAt.UTC(),
	}
	if err := s.rmqClient.PublishLocationUpdate(ctx, msg); err != nil {
		logger.Warn("UpdateLocation", "Failed to publish driver location", "", "", err.Error())
	}

	logger.Info("UpdateLocation", fmt.Sprintf("Driver %s location updated successfully", location.DriverID), "", "")
	return coord, nil
}

func (s *DriverService) Start(ctx context.Context, driverID uuid.UUID, rideId uuid.UUID, location model.Location) (dto.StartResponse, error) {
	logger.Info("Start", fmt.Sprintf("Driver %s attempting to start ride %s", driverID, rideId), "", string(rideId))

	status, err := s.repo.GetRideStatus(ctx, driverID, rideId)
	if err != nil {
		logger.Error("Start", "Failed to get ride status", "", string(rideId), err.Error())
		return dto.StartResponse{}, err
	}

	if status == model2.RideCompleted || status == model2.RideCancelled {
		logger.Warn("Start", "Ride already completed or cancelled", "", string(rideId), fmt.Sprintf("status: %s", status))
		return dto.StartResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}

	if location.Latitude < -90 || location.Latitude > 90 {
		logger.Warn("Start", "Invalid latitude", "", string(rideId), "latitude out of range")
		return dto.StartResponse{}, fmt.Errorf("latitude out of range")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		logger.Warn("Start", "Invalid longitude", "", string(rideId), "longitude out of range")
		return dto.StartResponse{}, fmt.Errorf("longitude out of range")
	}

	currentDStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error("Start", "Failed to get driver status", "", string(rideId), err.Error())
		return dto.StartResponse{}, err
	}

	if currentDStatus != "AVAILABLE" {
		logger.Warn("Start", "Driver is not available", "", string(rideId), fmt.Sprintf("status: %s", currentDStatus))
		return dto.StartResponse{}, errors.New("driver is not available")
	}

	newDStatus, startedAt, err := s.repo.Start(ctx, driverID, rideId, location)
	if err != nil {
		logger.Error("Start", "Failed to start ride", "", string(rideId), err.Error())
		return dto.StartResponse{}, err
	}

	resp := dto.StartResponse{
		RideID:    string(rideId),
		Status:    newDStatus,
		StartedAt: startedAt.Format(time.RFC3339),
		Message:   "Ride started successfully",
	}

	logger.Info("Start", fmt.Sprintf("Ride %s started by driver %s", rideId, driverID), "", string(rideId))
	return resp, nil
}

func (s *DriverService) Complete(ctx context.Context, driverID uuid.UUID, req dto.CompleteRequest) (dto.CompleteResponse, error) {
	logger.Info("Complete", fmt.Sprintf("Driver %s completing ride %s", driverID, req.RideID), "", string(req.RideID))

	status, err := s.repo.GetRideStatus(ctx, driverID, req.RideID)
	if err != nil {
		logger.Error("Complete", "Failed to get ride status", "", string(req.RideID), err.Error())
		return dto.CompleteResponse{}, err
	}

	if status == model2.RideCompleted || status == model2.RideCancelled {
		logger.Warn("Complete", "Ride already completed or cancelled", "", string(req.RideID), fmt.Sprintf("status: %s", status))
		return dto.CompleteResponse{}, fmt.Errorf("ride cannot be started (already completed or cancelled)")
	}

	const baseFare = 400.0
	const perKmRate = 120.0
	const perMinuteRate = 20.0

	driverEarnings := baseFare +
		req.ActualDistanceKm*perKmRate +
		req.ActualDurationMins*perMinuteRate

	if req.FinalLocation.Latitude < -90 || req.FinalLocation.Latitude > 90 {
		logger.Warn("Complete", "Invalid latitude", "", string(req.RideID), "latitude out of range")
		return dto.CompleteResponse{}, fmt.Errorf("latitude out of range")
	}
	if req.FinalLocation.Longitude < -180 || req.FinalLocation.Longitude > 180 {
		logger.Warn("Complete", "Invalid longitude", "", string(req.RideID), "longitude out of range")
		return dto.CompleteResponse{}, fmt.Errorf("longitude out of range")
	}
	if req.ActualDurationMins <= 0 {
		logger.Warn("Complete", "Invalid duration", "", string(req.RideID), "duration out of range")
		return dto.CompleteResponse{}, fmt.Errorf("duration out of range")
	}
	if req.ActualDistanceKm <= 0 {
		logger.Warn("Complete", "Invalid distance", "", string(req.RideID), "distance out of range")
		return dto.CompleteResponse{}, fmt.Errorf("duration out of range")
	}

	driverStatus, err := s.repo.GetDriverStatus(ctx, driverID)
	if err != nil {
		logger.Error("Complete", "Failed to get driver status", "", string(req.RideID), err.Error())
		return dto.CompleteResponse{}, err
	}
	if driverStatus != usermodel.DriverStatusBusy {
		logger.Warn("Complete", "Driver not busy", "", string(req.RideID), fmt.Sprintf("status: %s", driverStatus))
		return dto.CompleteResponse{}, errors.New("driver status not busy")
	}

	location := model.Location{
		Latitude:  req.FinalLocation.Latitude,
		Longitude: req.FinalLocation.Longitude,
	}
	completedAt, err := s.repo.Complete(ctx, driverID, driverEarnings, location, req.ActualDistanceKm, req.ActualDurationMins)
	if err != nil {
		logger.Error("Complete", "Failed to complete ride", "", string(req.RideID), err.Error())
		return dto.CompleteResponse{}, err
	}

	resp := dto.CompleteResponse{
		RideID:        string(req.RideID),
		Status:        usermodel.DriverStatusAvailable,
		CompletedAt:   completedAt.Format(time.RFC3339),
		DriverEarning: driverEarnings,
		Message:       "Ride completed successfully",
	}

	logger.Info("Complete", fmt.Sprintf("Ride %s completed by driver %s, earnings %.2f", req.RideID, driverID, driverEarnings), "", string(req.RideID))
	return resp, nil
}

func (s *DriverService) GetDriverInfo(ctx context.Context, driverID string) (model.DriverInfo, error) {
	logger.Info("GetDriverInfo", fmt.Sprintf("Fetching info for driver %s", driverID), "", "")
	response, err := s.repo.GetInfo(ctx, driverID)
	if err != nil {
		logger.Error("GetDriverInfo", "Failed to get driver info", "", "", err.Error())
		return model.DriverInfo{}, err
	}
	logger.Info("GetDriverInfo", fmt.Sprintf("Driver %s info retrieved successfully", driverID), "", "")
	return response, nil
}
