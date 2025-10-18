package rmq

import (
	"fmt"
	"log"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp.Connection
	Chan *amqp.Channel
	URL  string
}

func NewRabbitMQ(host string, port int, user, password string) (*RabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", user, password, host, port)
	rmq := &RabbitMQ{URL: url}

	if err := rmq.connect(); err != nil {
		return nil, err
	}
	return rmq, nil
}

func (r *RabbitMQ) connect() error {
	var conn *amqp.Connection
	var err error

	for i := 1; i <= 5; i++ {
		conn, err = amqp.Dial(r.URL)
		if err == nil {
			ch, chErr := conn.Channel()
			if chErr != nil {
				_ = conn.Close()
				return fmt.Errorf("failed to open channel: %w", chErr)
			}
			r.Conn = conn
			r.Chan = ch
			log.Println("Connected to RabbitMQ")
			return nil
		}

		log.Printf("RabbitMQ reconnect attempt %d failed: %v", i, err)
		time.Sleep(time.Second * time.Duration(math.Pow(2, float64(i)))) // exponential backoff
	}

	return fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
}

func (r *RabbitMQ) Close() {
	if r.Chan != nil {
		_ = r.Chan.Close()
	}
	if r.Conn != nil {
		_ = r.Conn.Close()
	}
	r.Conn, r.Chan = nil, nil
	log.Println("RabbitMQ connection closed")
}
