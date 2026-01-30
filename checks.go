package main

import (
	"context"
	"log"

	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func performHealthChecks(ctx context.Context, db *database.PostgresDB, rmq *worker.RabbitMQ) error {
	if err := db.HealthCheck(ctx); err != nil {
		log.Printf("✘ postgres health check falhou: %v", err)
		return err
	}
	log.Println("✔ postgres health check ok")

	if err := rmq.HealthCheck(); err != nil {
		log.Printf("✘ rabbitmq health check falhou: %v", err)
		return err
	}
	log.Println("✔ rabbitmq health check ok")

	return nil
}

func setupQueuesAndExchanges(rmq *worker.RabbitMQ) error {
	exchanges := []worker.ExchangeConfig{}

	queues := []worker.QueueConfig{}

	log.Println("sincronizando filas e exchanges...")

	queueNames := make([]string, len(queues))
	for i, q := range queues {
		queueNames[i] = q.Name
	}

	exchangeNames := make([]string, len(exchanges))
	for i, e := range exchanges {
		exchangeNames[i] = e.Name
	}

	syncConfig := worker.SyncConfig{
		Queues:    queueNames,
		Exchanges: exchangeNames,
		Prefix:    "",
	}

	if err := rmq.SyncQueuesAndExchanges(syncConfig); err != nil {
		log.Printf("aviso: não foi possível sincronizar (API de management indisponível?): %v", err)
	}

	for _, ex := range exchanges {
		if err := rmq.DeclareExchangeWithDefaults(ex.Name, ex.Kind); err != nil {
			return err
		}
		log.Printf("  ✔ exchange '%s' (%s) criado", ex.Name, ex.Kind)
	}

	for _, q := range queues {
		if err := rmq.DeclareQueueWithRetry(q); err != nil {
			return err
		}
		log.Printf("  ✔ fila '%s' criada com retry", q.Name)
	}

	return nil
}
