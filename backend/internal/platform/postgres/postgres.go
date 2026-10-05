package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

type Client struct {
	DB    *gorm.DB
	SQLDB *sql.DB
}

func New(cfg Config, logger *zap.Logger) (*Client, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"open postgres: %w",
			err,
		)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf(
			"get sql db: %w",
			err,
		)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf(
			"ping postgres: %w",
			err,
		)
	}
	logger.Info(
		"postgres connected",
		zap.Int(
			"max_open_conns",
			cfg.MaxOpenConns,
		),
		zap.Int(
			"max_idle_conns",
			cfg.MaxIdleConns,
		),
	)

	return &Client{
		DB:    db,
		SQLDB: sqlDB,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.SQLDB == nil {
		return nil
	}
	return c.SQLDB.Close()
}
