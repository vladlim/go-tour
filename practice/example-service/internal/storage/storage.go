package storage

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type storage struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

type Config struct {
	Username       string
	Password       string
	Host           string
	Database       string
	Port           int
	SSLMode        string
	MaxConns       int32
	ConnectTimeout time.Duration

	Logger *slog.Logger
}

func New(ctx context.Context, cfg Config) (*storage, error) {
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + cfg.Database,
		User:   url.UserPassword(cfg.Username, cfg.Password),
	}

	params := url.Values{}
	params.Set("sslmode", cfg.SSLMode)
	u.RawQuery = params.Encode()

	poolCfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	dbConn, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error init db connection: %w", err)
	}

	if err := dbConn.Ping(ctx); err != nil {
		dbConn.Close()
		return nil, fmt.Errorf("error ping db: %w", err)
	}

	return &storage{
		db:     dbConn,
		logger: cfg.Logger,
	}, nil
}

func (s *storage) Close() {
	s.db.Close()
}
