package rmq

import (
	"encoding/json"
	"fmt"

	"ride-hail-system/internal/common/logger"
	"ride-hail-system/internal/common/rmq"
)

func (c *Client) ConsumeDriverResponses(queueName string, handler func(msg rmq.DriverResponseMessage)) error {
	ch := c.Channel
	exchange := "driver_topic"

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("consume_driver_response", "failed to declare exchange", "", "", err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_driver_response", "failed to declare queue", "", "", err.Error())
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"driver.response.*",
		exchange,
		false,
		nil,
	); err != nil {
		logger.Error("consume_driver_response", "failed to bind queue", "", "", err.Error())
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	deliveries, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_driver_response", "failed to start consuming", "", "", err.Error())
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consume_driver_response", "started consuming driver responses", "", "")

	go func() {
		for d := range deliveries {
			var msg rmq.DriverResponseMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Warn("consume_driver_response", "failed to unmarshal driver response", "", "", err.Error())
				continue
			}
			logger.Debug("consume_driver_response", "received driver response message", "", msg.RideID)
			handler(msg)
		}
	}()

	return nil
}

func (c *Client) ConsumeDriverStatus(queueName string, handler func(msg rmq.RideStatusUpdateMessage)) error {
	ch := c.Channel
	exchange := "driver_topic"

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("consume_driver_status", "failed to declare exchange", "", "", err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_driver_status", "failed to declare queue", "", "", err.Error())
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"driver.status.*",
		exchange,
		false,
		nil,
	); err != nil {
		logger.Error("consume_driver_status", "failed to bind queue", "", "", err.Error())
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	deliveries, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_driver_status", "failed to start consuming", "", "", err.Error())
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consume_driver_status", "started consuming driver statuses", "", "")

	go func() {
		for d := range deliveries {
			var msg rmq.RideStatusUpdateMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Warn("consume_driver_status", "failed to unmarshal driver status", "", "", err.Error())
				continue
			}
			logger.Debug("consume_driver_status", "received driver status message", "", msg.RideID)
			handler(msg)
		}
	}()

	return nil
}

func (c *Client) ConsumeLocationUpdates(queueName string, handler func(msg rmq.LocationUpdateMessage)) error {
	ch := c.Channel
	exchange := "location_fanout"

	if err := ch.ExchangeDeclare(
		exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("consume_location", "failed to declare exchange", "", "", err.Error())
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_location", "failed to declare queue", "", "", err.Error())
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"", // fanout игнорирует routing key
		exchange,
		false,
		nil,
	); err != nil {
		logger.Error("consume_location", "failed to bind queue", "", "", err.Error())
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	deliveries, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("consume_location", "failed to start consuming", "", "", err.Error())
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	logger.Info("consume_location", "started consuming location updates", "", "")

	go func() {
		for d := range deliveries {
			var msg rmq.LocationUpdateMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Warn("consume_location", "failed to unmarshal location update", "", "", err.Error())
				continue
			}
			logger.Debug("consume_location", "received location update message", "", msg.RideID)
			handler(msg)
		}
	}()

	return nil
}
