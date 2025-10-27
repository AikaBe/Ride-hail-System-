package rmq

import (
	"encoding/json"
	"fmt"
	"ride-hail/internal/common/logger"
	"ride-hail/internal/common/rmq"
)

func (c *Client) ConsumeRideRequests(queueName string, handler func(msg rmq.RideRequestedMessage)) error {
	ch := c.Channel
	exchange := "ride_topic"

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("rmq_exchange_declare_failed", "Failed to declare exchange", exchange, "", err.Error(), "")
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	logger.Info("rmq_exchange_declared", "Exchange declared successfully", exchange, "")

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"ride.request.*",
		exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Начинаем потребление
	deliveries, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for d := range deliveries {
			var msg rmq.RideRequestedMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Warn("rmq_unmarshal_failed", "Failed to unmarshal ride request", queueName, "", err.Error())
				continue
			}
			logger.Info("rmq_message_received", "Ride request received", queueName, "")
			handler(msg)
		}
	}()

	return nil
}

func (c *Client) ConsumePassengerInfo(queueName string, handler func(msg rmq.PassiNFO)) error {
	ch := c.Channel
	exchange := "ride_topic"

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		logger.Error("rmq_exchange_declare_failed", "Failed to declare exchange", exchange, "", err.Error(), "")
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	logger.Info("rmq_exchange_declared", "Exchange declared successfully", exchange, "")

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"ride.request.*",
		exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Начинаем потребление
	deliveries, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for d := range deliveries {
			var msg rmq.PassiNFO
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Warn("rmq_unmarshal_failed", "Failed to unmarshal ride request", queueName, "", err.Error())
				continue
			}
			logger.Info("rmq_message_received", "Ride request received", queueName, "")
			handler(msg)
		}
	}()

	return nil
}

func (c *Client) ConsumeRideStatus(queueName string, handler func(msg rmq.RideStatusUpdateMessage)) error {
	ch := c.Channel
	exchange := "ride_topic"

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
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
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		q.Name,
		"ride.status.*",
		exchange,
		false,
		nil,
	); err != nil {
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
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for d := range deliveries {
			var msg rmq.RideStatusUpdateMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				fmt.Printf("WARN: failed to unmarshal ride status: %v\n", err)
				continue
			}
			handler(msg)
		}
	}()

	return nil
}
