package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/victor-lima-142/medical-service/pkg/config"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.WorkerMQConfig

	mu          sync.RWMutex
	closed      bool
	notifyClose chan *amqp.Error
}

func NewRabbitMQConnection(cfg *config.WorkerMQConfig) (*RabbitMQ, error) {
	rmq := &RabbitMQ{
		cfg: cfg,
	}

	if err := rmq.connect(); err != nil {
		return nil, err
	}

	return rmq, nil
}

func (r *RabbitMQ) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, err := amqp.Dial(r.cfg.URL())
	if err != nil {
		return fmt.Errorf("erro ao conectar com rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("erro ao abrir channel: %w", err)
	}

	if err := channel.Qos(10, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("erro ao configurar QoS: %w", err)
	}

	r.conn = conn
	r.channel = channel
	r.notifyClose = make(chan *amqp.Error)
	r.conn.NotifyClose(r.notifyClose)
	r.closed = false

	return nil
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channel
}

func (r *RabbitMQ) Connection() *amqp.Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn
}

func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true

	var errs []error

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("erro ao fechar channel: %w", err))
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("erro ao fechar conexão: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (r *RabbitMQ) Reconnect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-r.notifyClose:
			r.mu.RLock()
			closed := r.closed
			r.mu.RUnlock()

			if closed {
				return
			}

			log.Printf("conexão rabbitmq perdida: %v. Reconectando...", err)

			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(r.cfg.ReconnectDelay):
					if err := r.connect(); err != nil {
						log.Printf("falha ao reconectar: %v. Tentando novamente...", err)
						continue
					}
					log.Println("reconectado ao rabbitmq com sucesso")
					// Reiniciar loop de monitoramento
					goto continueMonitoring
				}
			}
		continueMonitoring:
		}
	}
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("canal não disponível")
	}

	return r.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		mandatory,
		immediate,
		msg,
	)
}

func (r *RabbitMQ) PublishJSON(ctx context.Context, exchange, routingKey string, body []byte) error {
	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	}

	return r.Publish(ctx, exchange, routingKey, false, false, msg)
}

func (r *RabbitMQ) PublishWithRetry(ctx context.Context, exchange, routingKey string, body []byte, retryCount int) error {
	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
		Headers: amqp.Table{
			"x-retry-count": int64(retryCount),
		},
	}

	return r.Publish(ctx, exchange, routingKey, false, false, msg)
}

func (r *RabbitMQ) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return nil, fmt.Errorf("canal não disponível")
	}

	return r.channel.Consume(
		queue,
		consumer,
		autoAck,
		exclusive,
		noLocal,
		noWait,
		args,
	)
}

func (r *RabbitMQ) HealthCheck() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("conexão rabbitmq fechada")
	}

	if r.channel == nil {
		return fmt.Errorf("channel rabbitmq não disponível")
	}

	return nil
}

func (r *RabbitMQ) IsClosed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.closed
}

func (r *RabbitMQ) Config() *config.WorkerMQConfig {
	return r.cfg
}
