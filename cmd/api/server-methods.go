package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/victor-lima-142/medical-service/pkg/config"
	"github.com/victor-lima-142/medical-service/pkg/config/database"
	"github.com/victor-lima-142/medical-service/pkg/config/worker"
)

func GetServer() *Server {
	return serverInstance
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) DB() *database.PostgresDB {
	return s.db
}

func (s *Server) RMQ() *worker.RabbitMQ {
	return s.rmq
}

func (s *Server) Config() *config.Config {
	return s.cfg
}

func getAPIPort() int {
	return 8080
}
