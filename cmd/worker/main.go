package worker

import (
	"context"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

type Worker struct {
	db       *database.PostgresDB
	rmq      *worker.RabbitMQ
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	handlers map[string]MessageHandler
}

type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

var workerInstance *Worker

func InitWorker(ctx context.Context, db *database.PostgresDB, rmq *worker.RabbitMQ) error {
	workerCtx, cancel := context.WithCancel(ctx)

	workerInstance = &Worker{
		db:       db,
		rmq:      rmq,
		ctx:      workerCtx,
		cancel:   cancel,
		handlers: make(map[string]MessageHandler),
	}

	workerInstance.registerHandlers()

	if err := workerInstance.startConsumers(); err != nil {
		cancel()
		return err
	}

	go func() {
		<-ctx.Done()
		workerInstance.Shutdown()
	}()

	return nil
}

func (w *Worker) registerHandlers() {
}

func (w *Worker) startConsumers() error {
	for queueName, handler := range w.handlers {
		if err := w.startConsumer(queueName, handler); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) startConsumer(queueName string, handler MessageHandler) error {
	deliveries, err := w.rmq.Consume(
		queueName,
		queueName+"-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.processMessages(queueName, deliveries, handler)
	}()

	log.Printf("  ✔ consumer iniciado para fila '%s'", queueName)
	return nil
}

func (w *Worker) processMessages(queueName string, deliveries <-chan amqp.Delivery, handler MessageHandler) {
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("consumer '%s' encerrado", queueName)
			return

		case _, ok := <-deliveries:
			if !ok {
				log.Printf("canal de entregas fechado para '%s'", queueName)
				return
			}

			//if err := w.processMessage(queueName, delivery, handler); err != nil {
			//	log.Printf("erro ao processar mensagem em '%s': %v", queueName, err)
			//	w.handleMessageError(delivery, err)
			// }
		}
	}
}

func (w *Worker) processMessage(_, delivery amqp.Delivery, handler MessageHandler) error {
	ctx, cancel := context.WithCancel(w.ctx)
	defer cancel()

	if err := handler(ctx, delivery); err != nil {
		return err
	}

	return delivery.Ack(false)
}

func (w *Worker) handleMessageError(delivery amqp.Delivery, err error) {
	retryCount := int64(0)
	if count, ok := delivery.Headers["x-retry-count"].(int64); ok {
		retryCount = count
	}

	maxRetries := int64(3)
	if retryCount >= maxRetries {
		log.Printf("mensagem enviada para DLQ após %d tentativas: %v", retryCount, err)
		delivery.Reject(false)
	} else {
		log.Printf("rejeitando mensagem para retry (tentativa %d/%d): %v", retryCount+1, maxRetries, err)
		delivery.Nack(false, true)
	}
}

func (w *Worker) Shutdown() {
	log.Println("encerrando worker...")
	w.cancel()
	w.wg.Wait()
	log.Println("worker encerrado")
}

func GetWorker() *Worker {
	return workerInstance
}

func (w *Worker) DB() *database.PostgresDB {
	return w.db
}

func (w *Worker) RMQ() *worker.RabbitMQ {
	return w.rmq
}
