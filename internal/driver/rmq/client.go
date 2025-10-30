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
	logger.Info("rmq_connect_request", "Connecting to RabbitMQ", rmqURL, "")
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		logger.Error("rmq_connect_failed", "Failed to connect to RabbitMQ", rmqURL, "", err.Error())
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	logger.Info("rmq_connect_success", "Connected to RabbitMQ", rmqURL, "")

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("rmq_channel_failed", "Failed to open channel", rmqURL, "", err.Error())
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	logger.Info("rmq_channel_success", "Channel opened successfully", rmqURL, "")

	return &Client{
		Conn:     conn,
		Channel:  ch,
		Exchange: exchange,
	}, nil
}

func (c *Client) Close() error {
	if c.Channel != nil {
		if err := c.Channel.Close(); err != nil {
			logger.Error("rmq_channel_close_failed", "Failed to close channel", c.Exchange, "", err.Error())
			return fmt.Errorf("failed to close channel: %w", err)
		}
		logger.Info("rmq_channel_closed", "Channel closed successfully", c.Exchange, "")
	}
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			logger.Error("rmq_conn_close_failed", "Failed to close connection", c.Exchange, "", err.Error())
			return fmt.Errorf("failed to close connection: %w", err)
		}
		logger.Info("rmq_conn_closed", "Connection closed successfully", c.Exchange, "")
	}
	return nil
}
