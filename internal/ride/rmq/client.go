package rmq

import (
	"fmt"

	"ride-hail/internal/common/logger"

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
		logger.Error("rmq_connect", "failed to connect to RabbitMQ", "", "", err.Error())
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	logger.Info("rmq_connect", "connected to RabbitMQ", "", "")

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("rmq_channel", "failed to open channel", "", "", err.Error())
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	logger.Info("rmq_channel", "channel opened successfully", "", "")

	return &Client{
		Conn:     conn,
		Channel:  ch,
		Exchange: exchange,
	}, nil
}

func (c *Client) Close() error {
	if c.Channel != nil {
		if err := c.Channel.Close(); err != nil {
			logger.Warn("rmq_close_channel", "failed to close channel", "", "", err.Error())
			return fmt.Errorf("failed to close channel: %w", err)
		}
		logger.Info("rmq_close_channel", "channel closed", "", "")
	}

	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			logger.Warn("rmq_close_connection", "failed to close connection", "", "", err.Error())
			return fmt.Errorf("failed to close connection: %w", err)
		}
		logger.Info("rmq_close_connection", "connection closed", "", "")
	}

	return nil
}
