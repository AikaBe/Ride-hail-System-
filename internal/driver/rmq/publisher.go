package rmq

import (
	"context"
	"encoding/json"
	"fmt"

	"ride-hail-system/internal/common/logger"
	"ride-hail-system/internal/common/rmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) PublishDriverResponse(ctx context.Context, msg rmq.DriverResponseMessage) error {
	logger.Info("publish_driver_response", "Preparing to publish driver response", "", msg.RideID)

	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_driver_response", "Failed to marshal driver response message", "", msg.RideID, err.Error())
		return err
	}

	routingKey := fmt.Sprintf("driver.response.%s", msg.RideID)

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		logger.Error("publish_driver_response", "Failed to declare exchange", "", msg.RideID, err.Error())
		return err
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
		logger.Error("publish_driver_response", "Failed to publish driver response", "", msg.RideID, err.Error())
		return err
	}

	logger.Info("publish_driver_response", "Driver response published successfully", "", msg.RideID)
	return nil
}

func (c *Client) PublishDriverStatus(ctx context.Context, msg rmq.RideStatusUpdateMessage) error {
	logger.Info("publish_driver_status", "Preparing to publish driver status", "", msg.RideID)

	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_driver_status", "Failed to marshal driver status message", "", msg.RideID, err.Error())
		return err
	}

	routingKey := fmt.Sprintf("driver.status.%s", msg.RideID)

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("publish_driver_status", "Failed to declare exchange", "", msg.RideID, err.Error())
		return err
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
		logger.Error("publish_driver_status", "Failed to publish driver status", "", msg.RideID, err.Error())
		return err
	}

	logger.Info("publish_driver_status", "Driver status published successfully", "", msg.RideID)
	return nil
}

func (c *Client) PublishLocationUpdate(ctx context.Context, msg rmq.LocationUpdateMessage) error {
	logger.Info("publish_location_update", "Preparing to publish driver location update", "", msg.DriverID)

	body, err := json.Marshal(msg)
	if err != nil {
		logger.Error("publish_location_update", "Failed to marshal location update message", "", msg.DriverID, err.Error())
		return err
	}

	exchange := "location_fanout"
	if err := c.Channel.ExchangeDeclare(
		exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("publish_location_update", "Failed to declare exchange", "", msg.DriverID, err.Error())
		return err
	}

	if err := c.Channel.PublishWithContext(
		ctx,
		exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		logger.Error("publish_location_update", "Failed to publish location update", "", msg.DriverID, err.Error())
		return err
	}

	logger.Info("publish_location_update", "Driver location update published successfully", "", msg.DriverID)
	return nil
}
