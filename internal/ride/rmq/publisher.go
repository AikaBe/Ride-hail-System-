package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"ride-hail/internal/common/rmq"
	"time"
)

func (c *Client) PublishRideRequested(ctx context.Context, msg rmq.RideRequestedMessage) error {
	if msg.CorrelationID == "" {
		msg.CorrelationID = generateCorrelationID()
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal ride request message: %w", err)
	}

	routingKey := fmt.Sprintf("ride.request.%s", msg.RideType)

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
		return fmt.Errorf("failed to publish ride request: %w", err)
	}

	return nil
}

func (c *Client) PublishLocationUpdate(ctx context.Context, msg rmq.LocationUpdateMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
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
		return fmt.Errorf("failed to publish location update: %w", err)
	}

	return nil
}

func generateCorrelationID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
