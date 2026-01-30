package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/victor-lima-142/medical-service/pkg/utils/environment"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	RabbitMQ WorkerMQConfig
}

type AppConfig struct {
	Env string
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
}

type WorkerMQConfig struct {
	Host           string
	Port           int
	User           string
	ManagementPort int
	Password       string
	VHost          string
	ReconnectDelay time.Duration
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if os.Getenv("APP_ENV") != "production" {
			return nil, fmt.Errorf("erro ao carregar .env: %w", err)
		}
	}

	cfg := &Config{
		App: AppConfig{
			Env: environment.GetEnv("APP_ENV", "development"),
		},
		Postgres: PostgresConfig{
			Host:            environment.GetEnv("POSTGRES_HOST", "localhost"),
			Port:            environment.GetEnvAsInt("POSTGRES_PORT", 5432),
			User:            environment.GetEnv("POSTGRES_USER", ""),
			Password:        environment.GetEnv("POSTGRES_PASSWORD", ""),
			Database:        environment.GetEnv("POSTGRES_DB", ""),
			SSLMode:         environment.GetEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns:        environment.GetEnvAsInt("POSTGRES_MAX_CONNS", 25),
			MinConns:        environment.GetEnvAsInt("POSTGRES_MIN_CONNS", 5),
			MaxConnLifetime: environment.GetEnvAsDuration("POSTGRES_MAX_CONN_LIFETIME", 5*time.Minute),
		},
		RabbitMQ: WorkerMQConfig{
			Host:           environment.GetEnv("RABBITMQ_HOST", "localhost"),
			Port:           environment.GetEnvAsInt("RABBITMQ_PORT", 5672),
			ManagementPort: environment.GetEnvAsInt("RABBITMQ_MANAGEMENT_PORT", 5672),
			User:           environment.GetEnv("RABBITMQ_DEFAULT_USER", ""),
			Password:       environment.GetEnv("RABBITMQ_DEFAULT_PASS", ""),
			VHost:          environment.GetEnv("RABBITMQ_VHOST", "/"),
			ReconnectDelay: environment.GetEnvAsDuration("RABBITMQ_RECONNECT_DELAY", 5*time.Second),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Postgres.User == "" {
		return fmt.Errorf("POSTGRES_USER é obrigatório")
	}
	if c.Postgres.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD é obrigatório")
	}
	if c.Postgres.Database == "" {
		return fmt.Errorf("POSTGRES_DB é obrigatório")
	}
	if c.RabbitMQ.User == "" {
		return fmt.Errorf("RABBITMQ_DEFAULT_USER é obrigatório")
	}
	if c.RabbitMQ.Password == "" {
		return fmt.Errorf("RABBITMQ_DEFAULT_PASS é obrigatório")
	}
	return nil
}

func (p *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode,
	)
}

func (r *WorkerMQConfig) URL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		r.User, r.Password, r.Host, r.Port, r.VHost,
	)
}

func (c *WorkerMQConfig) ManagementURL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.ManagementPort)
}
