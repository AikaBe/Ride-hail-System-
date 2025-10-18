package rmq

import (
	"encoding/json"
	"fmt"
	"ride-hail/internal/common/rmq"
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
		"driver.response.*",
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
			var msg rmq.DriverResponseMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				fmt.Printf("WARN: failed to unmarshal driver response: %v\n", err)
				continue
			}
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
		"driver.status.*",
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
				fmt.Printf("WARN: failed to unmarshal driver status: %v\n", err)
				continue
			}
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
		"", // fanout игнорирует routing key
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
			var msg rmq.LocationUpdateMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				fmt.Printf("WARN: failed to unmarshal location update: %v\n", err)
				continue
			}
			handler(msg)
		}
	}()

	return nil
}
