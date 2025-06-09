// backend/user-service/utils/publisher.go
package utils // Or a more specific package like 'eventpublisher'

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	rabbitConn    *amqp.Connection
	rabbitChannel *amqp.Channel
	rabbitOnce    sync.Once
	rabbitErr     error
)

func InitRabbitMQPublisher() {
	rabbitOnce.Do(func() {
		amqpURL := os.Getenv("RABBITMQ_URL")
		if amqpURL == "" {
			rabbitErr = fmt.Errorf("RABBITMQ_URL not set, publisher disabled")
			log.Println("Warning:", rabbitErr)
			return
		}

		conn, err := amqp.Dial(amqpURL)
		if err != nil {
			rabbitErr = fmt.Errorf("failed to connect to RabbitMQ: %w", err)
			log.Printf("Error connecting to RabbitMQ: %v. Publisher will be disabled.", err)
			return
		}
		rabbitConn = conn

		ch, err := rabbitConn.Channel()
		if err != nil {
			rabbitConn.Close()
			rabbitErr = fmt.Errorf("failed to open RabbitMQ channel: %w", err)
			log.Printf("Error opening RabbitMQ channel: %v. Publisher will be disabled.", err)
			return
		}
		rabbitChannel = ch

		err = rabbitChannel.ExchangeDeclare(
			"thread_events", // Example: for thread-related events
			"topic",         // type
			true,            // durable
			false,           // auto-deleted
			false,           // internal
			false,           // no-wait
			nil,             // arguments
		)
		if err != nil {
			log.Printf("Warning: Failed to declare exchange 'social_events': %v", err)
		}

		log.Println("RabbitMQ Publisher initialized successfully for thread-service.")

		go func() {
			<-rabbitConn.NotifyClose(make(chan *amqp.Error))
			log.Println("RabbitMQ connection closed. Publisher will attempt to reconnect or will be disabled.")
			rabbitChannel = nil
			rabbitConn = nil
			rabbitErr = errors.New("RabbitMQ connection lost")
		}()
	})
}

func PublishEvent(ctx context.Context, exchange, routingKey string, eventData interface{}) error {
	if rabbitChannel == nil || rabbitConn == nil || rabbitConn.IsClosed() {
		if rabbitErr == nil {
            rabbitErr = errors.New("RabbitMQ publisher not initialized or connection closed")
        }
		log.Printf("Skipping event publishing: %v", rabbitErr)
		return rabbitErr
	}

	body, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("Error marshalling event data for %s/%s: %v", exchange, routingKey, err)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = rabbitChannel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		})

	if err != nil {
		log.Printf("Error publishing event to %s/%s: %v", exchange, routingKey, err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("Event published to Exchange: '%s', RoutingKey: '%s', Body: %s", exchange, routingKey, string(body))
	return nil
}

func CloseRabbitMQPublisher() {
	if rabbitChannel != nil {
		rabbitChannel.Close()
	}
	if rabbitConn != nil {
		rabbitConn.Close()
	}
	log.Println("RabbitMQ Publisher connection closed.")
}