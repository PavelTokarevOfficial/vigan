package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"path/filepath"
	"time"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if e = p.Ping(c); e != nil {
		p.Close()
		return nil, e
	}
	return p, nil
}
func Migrate(url, dir string) error {
	m, e := migrate.New("file://"+filepath.Clean(dir), url)
	if e != nil {
		return e
	}
	defer m.Close()
	if e = m.Up(); e != nil && e != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", e)
	}
	return nil
}
func Health(ctx context.Context, p *pgxpool.Pool) error { return p.Ping(ctx) }
func LogStartup(l *slog.Logger)                         { l.Info("database ready") }

var _ = sql.ErrNoRows
