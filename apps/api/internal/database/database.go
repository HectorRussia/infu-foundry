package database

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Connection struct {
	ORM *gorm.DB
	SQL *sql.DB
}

// Open connects only when explicitly called. It never migrates any schema.
func Open(ctx context.Context, databaseURL string) (*Connection, error) {
	if err := ValidateURL(databaseURL); err != nil {
		return nil, err
	}
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: databaseURL,
		// Avoid session-bound statement caches when using a pooled Neon URL.
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.New("cannot initialize PostgreSQL; check the database URL")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, errors.New("cannot initialize PostgreSQL connection pool")
	}
	pool.SetMaxOpenConns(5)
	pool.SetMaxIdleConns(2)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, errors.New("cannot connect to PostgreSQL; check the database URL, TLS settings and network")
	}
	return &Connection{ORM: db, SQL: pool}, nil
}

func (c *Connection) Close() error { return c.SQL.Close() }

func ValidateURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" {
		return errors.New("database URL must be a postgres:// or postgresql:// URL with a host")
	}
	return nil
}
