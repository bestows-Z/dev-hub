package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const viewQueue = "devhub.page_views.v1"

type Broker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex
	events  chan ViewEvent
	logger  *zap.Logger
}

func NewBroker(cfg config.AnalyticsConfig, logger *zap.Logger) (*Broker, error) {
	uri := &url.URL{Scheme: "amqp", Host: cfg.RabbitAddr, Path: "/", User: url.UserPassword(cfg.RabbitUser, cfg.RabbitPassword)}
	conn, err := amqp.Dial(uri.String())
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := channel.QueueDeclare(viewQueue, true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Broker{conn: conn, channel: channel, events: make(chan ViewEvent, 512), logger: logger}, nil
}

func (b *Broker) Close() {
	_ = b.channel.Close()
	_ = b.conn.Close()
}

func (b *Broker) Record(kind, slug string) {
	event := ViewEvent{ID: uuid.NewString(), Kind: kind, Slug: slug, At: time.Now().UTC()}
	select {
	case b.events <- event:
	default:
		b.logger.Warn("analytics queue buffer full; skipping view")
	}
}

func (b *Broker) RunPublisher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-b.events:
			publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			if err := b.publish(publishCtx, event); err != nil {
				b.logger.Warn("publish page view", zap.Error(err))
			}
			cancel()
		}
	}
}

func (b *Broker) publish(ctx context.Context, event ViewEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.channel.PublishWithContext(ctx, "", viewQueue, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: data})
}

func (b *Broker) RunConsumer(ctx context.Context, store *Store) error {
	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := channel.Qos(10, 0, false); err != nil {
		return err
	}
	deliveries, err := channel.Consume(viewQueue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, open := <-deliveries:
			if !open {
				return fmt.Errorf("analytics consumer stopped")
			}
			var event ViewEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil || event.ID == "" {
				b.logger.Warn("discard invalid analytics event", zap.Error(err))
				_ = delivery.Ack(false)
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := store.Insert(writeCtx, event)
			cancel()
			if err != nil {
				b.logger.Warn("store page view", zap.Error(err))
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}
