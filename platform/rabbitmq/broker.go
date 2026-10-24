// Package rabbitmq wraps connecting, publishing and consuming events
// on a shared "topic" exchange.
package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	amqp "github.com/rabbitmq/amqp091-go"
)

const exchange = "ecommerce.events"

var ErrConnectionLost = errors.New("rabbitmq connection lost")

// consumed makes a consumer stuck in a retry loop visible (outcome="retried").
var consumed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "events_consumed_total",
	Help: "Consumed events by routing key and outcome (acked, dropped, retried).",
}, []string{"routing_key", "outcome"})

// Handler processes a message body. An error wrapped with Permanent drops
// the message; any other error requeues it for another attempt.
type Handler func(ctx context.Context, body []byte) error

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent marks an error that retrying won't fix (unreadable message…).
func Permanent(err error) error { return permanentError{err: err} }

type Broker struct {
	conn *amqp.Connection

	mu  sync.Mutex // an AMQP channel must not publish from several goroutines at once
	pub *amqp.Channel
}

func Connect(ctx context.Context, url string) (*Broker, error) {
	conn, err := dialWithRetry(ctx, url)
	if err != nil {
		return nil, err
	}

	pub, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := pub.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return &Broker{conn: conn, pub: pub}, nil
}

func dialWithRetry(ctx context.Context, url string) (*amqp.Connection, error) {
	const attempts = 15
	var err error
	for i := 1; i <= attempts; i++ {
		var conn *amqp.Connection
		if conn, err = amqp.Dial(url); err == nil {
			return conn, nil
		}
		log.Printf("rabbitmq not ready (attempt %d/%d): %v", i, attempts, err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("connect to rabbitmq: %w", err)
}

func (b *Broker) Close() error {
	return b.conn.Close()
}

// CancelOnConnectionLost cancels the service context if the connection drops:
// the process exits and the orchestrator restarts it cleanly, instead of
// running on without consuming events.
func (b *Broker) CancelOnConnectionLost(cancel context.CancelCauseFunc) {
	closed := b.conn.NotifyClose(make(chan *amqp.Error, 1))
	go func() {
		if amqpErr, ok := <-closed; ok && amqpErr != nil {
			cancel(fmt.Errorf("%w: %v", ErrConnectionLost, amqpErr))
		}
	}()
}

func (b *Broker) Publish(ctx context.Context, routingKey string, event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pub.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// Subscribe creates a durable queue bound to the given routing keys and
// processes messages in the background until ctx is cancelled.
func (b *Broker) Subscribe(ctx context.Context, queue string, routingKeys []string, handle Handler) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	for _, key := range routingKeys {
		if err := ch.QueueBind(queue, key, exchange, false, nil); err != nil {
			return err
		}
	}
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		defer ch.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				dispatch(ctx, msg, handle)
			}
		}
	}()
	return nil
}

func dispatch(ctx context.Context, msg amqp.Delivery, handle Handler) {
	err := handle(ctx, msg.Body)

	var permanent permanentError
	switch {
	case err == nil:
		consumed.WithLabelValues(msg.RoutingKey, "acked").Inc()
		_ = msg.Ack(false)
	case errors.As(err, &permanent):
		consumed.WithLabelValues(msg.RoutingKey, "dropped").Inc()
		log.Printf("dropping message %q: %v", msg.RoutingKey, err)
		_ = msg.Nack(false, false)
	default:
		consumed.WithLabelValues(msg.RoutingKey, "retried").Inc()
		log.Printf("handling %q failed, will retry: %v", msg.RoutingKey, err)
		time.Sleep(time.Second) // avoids spinning at full speed while a dependency is down
		_ = msg.Nack(false, true)
	}
}
