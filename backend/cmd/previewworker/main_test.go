package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/runtimejobs"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWorkerClaimsQueuedJobIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_RUNTIME_WORKER") != "1" {
		t.Skip("set DEVHUB_TEST_RUNTIME_WORKER=1 to test PostgreSQL worker queue")
	}
	cfg := config.PostgresConfig{Host: "127.0.0.1", Port: 5432, User: os.Getenv("POSTGRES_USER"), Password: os.Getenv("POSTGRES_PASSWORD"), DBName: os.Getenv("POSTGRES_DB"), SSLMode: "disable", TimeZone: "UTC"}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var queued int64
	if err := db.Model(&runtimejobs.Job{}).Where("status = ?", "queued").Count(&queued).Error; err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Skip("another runtime job is queued")
	}
	suffix := time.Now().UnixNano()
	u := user.User{Username: fmt.Sprintf("worker_%d", suffix), Email: fmt.Sprintf("worker_%d@example.invalid", suffix), PasswordHash: "test-only", Role: user.RoleAdmin, Status: user.StatusNormal}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&u)
	p := project.Project{Slug: fmt.Sprintf("worker-test-%d", suffix), Title: "Worker test", Tags: []string{}, Status: "draft", RuntimeStatus: "stopped"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&p)
	job := runtimejobs.Job{ProjectID: p.ID, Action: "stop", Status: "queued", RequestedBy: u.ID}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&job)
	called := false
	if err := runOnce(context.Background(), db, func(_ context.Context, action, slug string) error {
		called = true
		if action != "stop" || slug != p.Slug {
			return fmt.Errorf("unexpected call %s %s", action, slug)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&job, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !called || job.Status != "succeeded" || job.StartedAt == nil || job.FinishedAt == nil {
		t.Fatalf("worker did not complete job: %+v", job)
	}
}
