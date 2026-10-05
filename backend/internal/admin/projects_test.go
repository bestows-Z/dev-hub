package admin

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestExistingBundlePreviewSurvivesProjectEdit(t *testing.T) {
	input := projectInput{Slug: "new-slug", Title: "Project", PreviewURL: "/api/v1/project-previews/old-slug/index.html"}
	if !validProjectInput(input) {
		t.Fatal("a project with an uploaded bundle must remain editable after its slug changes")
	}
}

func TestCreateProjectAgainstMigratedPostgres(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_POSTGRES") != "1" {
		t.Skip("set DEVHUB_TEST_POSTGRES=1 to run the PostgreSQL integration test")
	}
	_ = godotenv.Load("../../../.env")
	t.Setenv("APP_ENV", "test")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	client, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	slug := fmt.Sprintf("runtime-create-test-%d", time.Now().UnixNano())
	rollback := errors.New("rollback test project")
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/projects", NewHandler(tx, nil, zap.NewNop()).CreateProject)
		request := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(fmt.Sprintf(`{"slug":%q,"title":"Runtime test"}`, slug)))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			return fmt.Errorf("create project status %d: %s", response.Code, response.Body.String())
		}
		var item project.Project
		if err := tx.Where("slug = ?", slug).First(&item).Error; err != nil {
			return err
		}
		if item.RuntimeStatus != "stopped" {
			return fmt.Errorf("runtime_status = %q", item.RuntimeStatus)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
