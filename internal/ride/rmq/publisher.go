package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/rmq"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) PublishRideRequested(ctx context.Context, msg rmq.RideRequestedMessage) error {
	if msg.CorrelationID == "" {
		msg.CorrelationID = generateCorrelationID()
	}

	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_ride_requested", "failed to marshal ride request message", msg.CorrelationID, msg.RideID, err.Error())
		return fmt.Errorf("failed to marshal ride request message: %w", err)
	}

	routingKey := fmt.Sprintf("ride.request.%s", msg.RideType)

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("publish_ride_requested", "failed to declare exchange", msg.CorrelationID, msg.RideID, err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	if err := c.Channel.PublishWithContext(
		ctx,
		c.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		logger.Error("publish_ride_requested", "failed to publish ride request", msg.CorrelationID, msg.RideID, err.Error())
		return fmt.Errorf("failed to publish ride request: %w", err)
	}

	logger.Info("publish_ride_requested", "ride request successfully published", msg.CorrelationID, msg.RideID)
	return nil
}

func (c *Client) PublishPassengerInfo(ctx context.Context, msg rmq.PassiNFO) error {
	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_passenger_info", "failed to marshal passenger info message", "", "", err.Error())
		return fmt.Errorf("failed to marshal passenger info message: %w", err)
	}

	routingKey := fmt.Sprintf("ride.request.%s", msg.Type)

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("publish_passenger_info", "failed to declare exchange", "", "", err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	if err := c.Channel.PublishWithContext(
		ctx,
		c.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		logger.Error("publish_passenger_info", "failed to publish passenger info", "", "", err.Error())
		return fmt.Errorf("failed to publish passenger info: %w", err)
	}

	logger.Info("publish_passenger_info", "passenger info successfully published", "", "")
	return nil
}

func (c *Client) PublishLocationUpdate(ctx context.Context, msg rmq.LocationUpdateMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_location_update", "failed to marshal location update message", "", msg.RideID, err.Error())
		return fmt.Errorf("failed to marshal location update: %w", err)
	}

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("publish_location_update", "failed to declare exchange", "", msg.RideID, err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	if err := c.Channel.PublishWithContext(
		ctx,
		c.Exchange,
		"", // fanout ignores routing key
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		logger.Error("publish_location_update", "failed to publish location update", "", msg.RideID, err.Error())
		return fmt.Errorf("failed to publish location update: %w", err)
	}

	logger.Debug("publish_location_update", "location update successfully published", "", msg.RideID)
	return nil
}

func generateCorrelationID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
