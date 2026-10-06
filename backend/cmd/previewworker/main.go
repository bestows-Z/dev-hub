package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/runtimejobs"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type runner func(context.Context, string, string) error

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}
	connection, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, logger)
	if err != nil {
		logger.Fatal("connect database", zap.Error(err))
	}
	defer connection.Close()
	bin := os.Getenv("PREVIEW_RUNNER_BIN")
	if bin == "" {
		bin = "./bin/devhub-preview"
	}
	if _, err := os.Stat(bin); err != nil {
		logger.Fatal("preview runner binary is missing", zap.String("path", bin), zap.Error(err))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	call := func(ctx context.Context, action, slug string) error {
		command := "deploy-zip"
		if action == "stop" {
			command = "stop"
		}
		output, err := exec.CommandContext(ctx, bin, command, "--slug", slug).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %s: %w: %s", command, slug, err, truncate(string(output), 1000))
		}
		return nil
	}
	logger.Info("project preview worker ready")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := failExpiredJobs(connection.DB); err != nil {
			logger.Error("expire abandoned jobs", zap.Error(err))
		}
		if err := runOnce(ctx, connection.DB, call); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("runtime job", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func failExpiredJobs(db *gorm.DB) error {
	return db.Model(&runtimejobs.Job{}).Where("status = ? AND started_at < ?", "running", time.Now().Add(-20*time.Minute)).
		Updates(map[string]any{"status": "failed", "error_text": "worker stopped before completing this job", "finished_at": time.Now()}).Error
}

func runOnce(ctx context.Context, db *gorm.DB, call runner) error {
	var job runtimejobs.Job
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status = ?", "queued").Order("created_at ASC, id ASC").Limit(1).Find(&job)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		now := time.Now()
		return tx.Model(&job).Updates(map[string]any{"status": "running", "started_at": now}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("claim job: %w", err)
	}
	var item project.Project
	if err := db.WithContext(ctx).First(&item, job.ProjectID).Error; err != nil {
		return finishJob(db, job, fmt.Errorf("find project: %w", err))
	}
	duration := 16 * time.Minute
	if job.Action == "stop" {
		duration = 3 * time.Minute
	}
	jobCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	runErr := call(jobCtx, job.Action, item.Slug)
	if err := finishJob(db, job, runErr); err != nil {
		return err
	}
	return runErr
}

func finishJob(db *gorm.DB, job runtimejobs.Job, runErr error) error {
	status, detail := "succeeded", ""
	if runErr != nil {
		status, detail = "failed", truncate(runErr.Error(), 1000)
	}
	return db.Model(&job).Updates(map[string]any{"status": status, "error_text": detail, "finished_at": time.Now()}).Error
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}
