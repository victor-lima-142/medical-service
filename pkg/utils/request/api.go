package request

import (
	"github.com/gin-gonic/gin"
	"github.com/victor-lima-142/medical-service/pkg/config"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func GetDB(c *gin.Context) *database.PostgresDB {
	db, exists := c.Get("db")
	if !exists {
		return nil
	}
	return db.(*database.PostgresDB)
}

func GetRMQ(c *gin.Context) *worker.RabbitMQ {
	rmq, exists := c.Get("rmq")
	if !exists {
		return nil
	}
	return rmq.(*worker.RabbitMQ)
}

func GetConfig(c *gin.Context) *config.Config {
	cfg, exists := c.Get("cfg")
	if !exists {
		return nil
	}
	return cfg.(*config.Config)
}

func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get("request_id")
	if !exists {
		return ""
	}
	return requestID.(string)
}
