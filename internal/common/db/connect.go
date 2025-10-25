package db

import (
	"context"
	"fmt"
	"ride-hail/internal/common/logger"
	"time"

	"github.com/jackc/pgx/v5"
)

type Postgres struct {
	Conn *pgx.Conn
}

func NewPostgres(host string, port int, user, password, database string) (*Postgres, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, database,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		logger.Error("db_connection_failed", "Failed to connect to Postgres", "", "", err.Error(), "")
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		logger.Error("db_ping_failed", "Postgres ping failed", "", "", err.Error(), "")
		conn.Close(ctx)
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	logger.Info("db_connected", "Connected to PostgreSQL successfully", "", "")
	return &Postgres{Conn: conn}, nil
}

func (p *Postgres) Close() {
	if p.Conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		p.Conn.Close(ctx)
		logger.Info("db_connection_closed", "PostgreSQL connection closed", "", "")
	}
}
