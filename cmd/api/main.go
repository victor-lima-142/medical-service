package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/victor-lima-142/medical-service/pkg/config"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func InitAPI(_ context.Context, db *database.PostgresDB, rmq *worker.RabbitMQ, cfg *config.Config) (*http.Server, error) {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(corsMiddleware())
	router.Use(requestIDMiddleware())

	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("rmq", rmq)
		c.Set("cfg", cfg)
		c.Next()
	})

	registerRoutes(router, db, rmq)

	port := getAPIPort()
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverInstance = &Server{
		httpServer: httpServer,
		router:     router,
		db:         db,
		rmq:        rmq,
		cfg:        cfg,
	}

	go func() {
		log.Printf("servidor HTTP iniciando na porta %d", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("erro no servidor HTTP: %v", err)
		}
	}()

	return httpServer, nil
}

func registerRoutes(router *gin.Engine, db *database.PostgresDB, rmq *worker.RabbitMQ) {
	health := router.Group("/health")
	{
		health.GET("", healthHandler(db, rmq))
		health.GET("/live", livenessHandler())
		health.GET("/ready", readinessHandler(db, rmq))
	}

	v1 := router.Group("/api/v1")
	{
		v1.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "API v1 - Medical Service",
				"version": "1.0.0",
			})
		})
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Rota não encontrada",
		})
	})
}
