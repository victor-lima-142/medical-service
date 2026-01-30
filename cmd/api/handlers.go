package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func healthHandler(db *database.PostgresDB, rmq *worker.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		dbStatus := "healthy"
		dbHealthy := true
		if err := db.HealthCheck(ctx); err != nil {
			dbStatus = fmt.Sprintf("unhealthy: %v", err)
			dbHealthy = false
		}

		rmqStatus := "healthy"
		rmqHealthy := true
		if err := rmq.HealthCheck(); err != nil {
			rmqStatus = fmt.Sprintf("unhealthy: %v", err)
			rmqHealthy = false
		}

		status := http.StatusOK
		overallStatus := "healthy"
		if !dbHealthy || !rmqHealthy {
			status = http.StatusServiceUnavailable
			overallStatus = "unhealthy"
		}

		c.JSON(status, gin.H{
			"status":    overallStatus,
			"database":  dbStatus,
			"rabbitmq":  rmqStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func livenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func readinessHandler(db *database.PostgresDB, rmq *worker.RabbitMQ) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := db.HealthCheck(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": fmt.Sprintf("database: %v", err),
			})
			return
		}

		if err := rmq.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": fmt.Sprintf("rabbitmq: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
