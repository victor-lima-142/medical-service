package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victor-lima-142/medical-service/pkg/config"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
	cfg  *config.PostgresConfig
}

func NewPostgresConnection(ctx context.Context, cfg *config.PostgresConfig) (*PostgresDB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear config do postgres: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar pool de conexões: %w", err)
	}

	db := &PostgresDB{
		Pool: pool,
		cfg:  cfg,
	}

	if err := db.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("erro ao conectar com postgres: %w", err)
	}

	return db, nil
}

func (db *PostgresDB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.Pool.Ping(ctx)
}

func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	var result int
	err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check falhou: %w", err)
	}
	return nil
}

func (db *PostgresDB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
