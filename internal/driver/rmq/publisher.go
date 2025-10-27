package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/common/rmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) PublishDriverResponse(ctx context.Context, msg rmq.DriverResponseMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal driver response message: %w", err)
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
		return fmt.Errorf("failed to publish driver response: %w", err)
	}

	return nil
}

func (c *Client) PublishDriverStatus(ctx context.Context, msg rmq.RideStatusUpdateMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal driver status message: %w", err)
	}

	routingKey := fmt.Sprintf("driver.status.%s", msg.RideID) // или DriverID, если нужно по водителю

	if err := c.Channel.ExchangeDeclare(
		c.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
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
		return fmt.Errorf("failed to publish driver status: %w", err)
	}

	return nil
}

func (c *Client) PublishLocationUpdate(ctx context.Context, msg rmq.LocationUpdateMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal location update message: %w", err)
	}

	if err := c.Channel.ExchangeDeclare(
		"location_fanout",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	if err := c.Channel.PublishWithContext(
		ctx,
		c.Exchange,
		"", // routing key игнорируется для fanout
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return fmt.Errorf("failed to publish location update: %w", err)
	}

	return nil
}
