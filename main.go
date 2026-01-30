package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victor-lima-142/medical-service/cmd/api"
	cmdWorker "github.com/victor-lima-142/medical-service/cmd/worker"
	"github.com/victor-lima-142/medical-service/pkg/config"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("erro ao carregar configurações: %v", err)
	}
	log.Printf("ambiente: %s", cfg.App.Env)

	db, err := database.NewPostgresConnection(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalf("erro ao conectar com postgres: %v", err)
	}
	defer db.Close()
	log.Println("✔ conectado ao postgresql")

	rmq, err := worker.NewRabbitMQConnection(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("erro ao conectar com rabbitmq: %v", err)
	}
	defer rmq.Close()
	log.Println("✔ conectado ao rabbitmq")

	go rmq.Reconnect(ctx)

	if err := performHealthChecks(ctx, db, rmq); err != nil {
		log.Fatalf("health check inicial falhou: %v", err)
	}

	if err := setupQueuesAndExchanges(rmq); err != nil {
		log.Fatalf("erro ao configurar filas e exchanges: %v", err)
	}
	log.Println("✔ filas e exchanges configurados")

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	if err := cmdWorker.InitWorker(workerCtx, db, rmq); err != nil {
		log.Fatalf("erro ao inicializar worker: %v", err)
	}
	log.Println("✔ worker inicializado")

	if err := performHealthChecks(ctx, db, rmq); err != nil {
		log.Printf("aviso: health check pós-inicialização falhou: %v", err)
	}

	apiServer, err := api.InitAPI(ctx, db, rmq, cfg)
	if err != nil {
		log.Fatalf("erro ao inicializar API: %v", err)
	}
	log.Println("✔ API inicializada")

	stats := db.Stats()
	log.Printf("postgres pool stats: total=%d, idle=%d, in_use=%d",
		stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())

	log.Println("✔ aplicação iniciada com sucesso")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("encerrando aplicação...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if apiServer != nil {
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("erro ao encerrar servidor HTTP: %v", err)
		}
	}

	workerCancel()
	cancel()

	log.Println("aplicação encerrada com sucesso")
}
