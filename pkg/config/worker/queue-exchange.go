package worker

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueConfig struct {
	Name       string
	RoutingKey string
	Exchange   string
}

type ExchangeConfig struct {
	Name string
	Kind string
}

type RetryConfig struct {
	MaxRetries    int
	RetryDelays   []time.Duration
	DLXExchange   string
	DLXRoutingKey string
}

func DefaultRetryConfig(queueName string) RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		RetryDelays: []time.Duration{
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
		},
		DLXExchange:   "medical.dlx",
		DLXRoutingKey: queueName + ".dead",
	}
}

type QueueCreation struct {
	name       string
	durable    *bool
	autoDelete *bool
	exclusive  *bool
	noWait     *bool
	args       amqp.Table
}

func CreateQueue(name string, durable, autoDelete, exclusive, noWait *bool, args *amqp.Table) *QueueCreation {
	var arguments amqp.Table
	if args != nil {
		arguments = *args
	}
	return &QueueCreation{
		name:       name,
		durable:    durable,
		autoDelete: autoDelete,
		exclusive:  exclusive,
		noWait:     noWait,
		args:       arguments,
	}
}

func CreateSimpleQueue(name string, args *amqp.Table) *QueueCreation {
	var arguments amqp.Table
	if args != nil {
		arguments = *args
	}
	return &QueueCreation{
		name:       name,
		durable:    nil,
		autoDelete: nil,
		exclusive:  nil,
		noWait:     nil,
		args:       arguments,
	}
}

func CreateDurableQueue(name string, args amqp.Table) *QueueCreation {
	durable := true
	autoDelete := false
	exclusive := false
	noWait := false

	return &QueueCreation{
		name:       name,
		durable:    &durable,
		autoDelete: &autoDelete,
		exclusive:  &exclusive,
		noWait:     &noWait,
		args:       args,
	}
}

type ExchangeCreation struct {
	name       string
	kind       string
	durable    *bool
	autoDelete *bool
	internal   *bool
	noWait     *bool
	args       amqp.Table
}

func CreateExchange(name, kind string, durable, autoDelete, internal, noWait *bool, args *amqp.Table) *ExchangeCreation {
	var arguments amqp.Table
	if args != nil {
		arguments = *args
	}
	return &ExchangeCreation{
		name:       name,
		kind:       kind,
		durable:    durable,
		autoDelete: autoDelete,
		internal:   internal,
		noWait:     noWait,
		args:       arguments,
	}
}

func CreateSimpleExchange(name, kind string, args *amqp.Table) *ExchangeCreation {
	var arguments amqp.Table
	if args != nil {
		arguments = *args
	}
	return &ExchangeCreation{
		name:       name,
		kind:       kind,
		durable:    nil,
		autoDelete: nil,
		internal:   nil,
		noWait:     nil,
		args:       arguments,
	}
}

func CreateDurableExchange(name, kind string, args amqp.Table) *ExchangeCreation {
	durable := true
	autoDelete := false
	internal := false
	noWait := false

	return &ExchangeCreation{
		name:       name,
		kind:       kind,
		durable:    &durable,
		autoDelete: &autoDelete,
		internal:   &internal,
		noWait:     &noWait,
		args:       args,
	}
}

func (r *RabbitMQ) DeclareQueue(queueCreation QueueCreation) (amqp.Queue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	durable := true
	autoDelete := false
	exclusive := false
	noWait := false

	if queueCreation.durable != nil {
		durable = *queueCreation.durable
	}
	if queueCreation.autoDelete != nil {
		autoDelete = *queueCreation.autoDelete
	}
	if queueCreation.exclusive != nil {
		exclusive = *queueCreation.exclusive
	}
	if queueCreation.noWait != nil {
		noWait = *queueCreation.noWait
	}

	return r.channel.QueueDeclare(
		queueCreation.name,
		durable,
		autoDelete,
		exclusive,
		noWait,
		queueCreation.args,
	)
}

func (r *RabbitMQ) DeclareExchange(exchangeCreation ExchangeCreation) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	durable := true
	autoDelete := false
	internal := false
	noWait := false

	if exchangeCreation.durable != nil {
		durable = *exchangeCreation.durable
	}
	if exchangeCreation.autoDelete != nil {
		autoDelete = *exchangeCreation.autoDelete
	}
	if exchangeCreation.internal != nil {
		internal = *exchangeCreation.internal
	}
	if exchangeCreation.noWait != nil {
		noWait = *exchangeCreation.noWait
	}

	return r.channel.ExchangeDeclare(
		exchangeCreation.name,
		exchangeCreation.kind,
		durable,
		autoDelete,
		internal,
		noWait,
		exchangeCreation.args,
	)
}

func (r *RabbitMQ) DeclareExchangeWithDefaults(name, kind string) error {
	exchange := CreateDurableExchange(name, kind, nil)
	return r.DeclareExchange(*exchange)
}

func (r *RabbitMQ) DeclareQueueWithRetry(config QueueConfig) error {
	retryConfig := DefaultRetryConfig(config.Name)

	dlqName := config.Name + ".dlq"
	dlqArgs := amqp.Table{
		"x-queue-type": "classic",
	}

	dlq := CreateDurableQueue(dlqName, dlqArgs)
	if _, err := r.DeclareQueue(*dlq); err != nil {
		return fmt.Errorf("erro ao criar DLQ '%s': %w", dlqName, err)
	}

	if err := r.BindQueue(dlqName, retryConfig.DLXRoutingKey, retryConfig.DLXExchange, false, nil); err != nil {
		return fmt.Errorf("erro ao fazer bind da DLQ '%s': %w", dlqName, err)
	}

	for i, delay := range retryConfig.RetryDelays {
		retryQueueName := fmt.Sprintf("%s.retry.%d", config.Name, i+1)

		retryArgs := amqp.Table{
			"x-dead-letter-exchange":    config.Exchange,
			"x-dead-letter-routing-key": config.RoutingKey,
			"x-message-ttl":             int64(delay.Milliseconds()),
			"x-queue-type":              "classic",
		}

		retryQueue := CreateDurableQueue(retryQueueName, retryArgs)
		if _, err := r.DeclareQueue(*retryQueue); err != nil {
			return fmt.Errorf("erro ao criar fila de retry '%s': %w", retryQueueName, err)
		}
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    retryConfig.DLXExchange,
		"x-dead-letter-routing-key": retryConfig.DLXRoutingKey,
		"x-queue-type":              "classic",
	}

	mainQueue := CreateDurableQueue(config.Name, mainArgs)
	if _, err := r.DeclareQueue(*mainQueue); err != nil {
		return fmt.Errorf("erro ao criar fila principal '%s': %w", config.Name, err)
	}

	if config.Exchange != "" && config.RoutingKey != "" {
		if err := r.BindQueue(config.Name, config.RoutingKey, config.Exchange, false, nil); err != nil {
			return fmt.Errorf("erro ao fazer bind da fila '%s': %w", config.Name, err)
		}
	}

	return nil
}

func (r *RabbitMQ) DeclareSimpleQueueDurable(name string) (amqp.Queue, error) {
	queue := CreateDurableQueue(name, nil)
	return r.DeclareQueue(*queue)
}

func (r *RabbitMQ) BindQueue(queueName, routingKey, exchangeName string, noWait bool, args amqp.Table) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.channel.QueueBind(
		queueName,
		routingKey,
		exchangeName,
		noWait,
		args,
	)
}

func (r *RabbitMQ) UnbindQueue(queueName, routingKey, exchangeName string, args amqp.Table) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.channel.QueueUnbind(
		queueName,
		routingKey,
		exchangeName,
		args,
	)
}

func (r *RabbitMQ) DeleteQueue(name string, ifUnused, ifEmpty bool) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.channel.QueueDelete(name, ifUnused, ifEmpty, false)
}

func (r *RabbitMQ) DeleteExchange(name string, ifUnused bool) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.channel.ExchangeDelete(name, ifUnused, false)
}

func (r *RabbitMQ) PurgeQueue(name string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.channel.QueuePurge(name, false)
}

func GetRetryQueueName(baseName string, retryLevel int) string {
	return fmt.Sprintf("%s.retry.%d", baseName, retryLevel)
}

func GetDLQName(baseName string) string {
	return baseName + ".dlq"
}
