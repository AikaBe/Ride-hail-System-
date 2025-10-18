package rmq

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	Exchange string
}

func NewClient(rmqURL, exchange string) (*Client, error) {
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &Client{
		Conn:     conn,
		Channel:  ch,
		Exchange: exchange,
	}, nil
}

func (c *Client) Close() error {
	if c.Channel != nil {
		if err := c.Channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}
