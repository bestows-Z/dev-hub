package admin

import (
	"bytes"
	"encoding/json"
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
	"github.com/bestows-Z/dev-hub/backend/internal/runtimejobs"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
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

func TestQueueRuntimeJobAgainstPostgres(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_POSTGRES") != "1" {
		t.Skip("set DEVHUB_TEST_POSTGRES=1 to test runtime queue")
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
	rollback := errors.New("rollback runtime queue test")
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		suffix := time.Now().UnixNano()
		u := user.User{Username: fmt.Sprintf("runtime_%d", suffix), Email: fmt.Sprintf("runtime_%d@example.invalid", suffix), PasswordHash: "test-only", Role: user.RoleAdmin, Status: user.StatusNormal}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		item := project.Project{Slug: fmt.Sprintf("queue-test-%d", suffix), Title: "Queue test", Tags: []string{}, Status: "draft", RuntimeStatus: "stopped", RuntimeBundleKey: "test/runtime.zip", RuntimeBundleUploaded: true}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set("currentUser", &u); c.Next() })
		handler := NewHandler(tx, nil, zap.NewNop())
		router.POST("/projects/:id/runtime-jobs", handler.QueueProjectRuntime)
		router.GET("/projects", handler.ListProjects)
		request := func() *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%d/runtime-jobs", item.ID), bytes.NewBufferString(`{"action":"start"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		}
		if got := request(); got.Code != http.StatusOK {
			return fmt.Errorf("queue returned %d: %s", got.Code, got.Body.String())
		}
		var job runtimejobs.Job
		if err := tx.Where("project_id = ?", item.ID).Take(&job).Error; err != nil {
			return err
		}
		if job.Action != "start" || job.Status != "queued" || job.RequestedBy != u.ID {
			return fmt.Errorf("bad job: %+v", job)
		}
		list := httptest.NewRecorder()
		router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/projects?q="+item.Slug, nil))
		if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"runtime_job_status":"queued"`)) {
			return fmt.Errorf("project row missing runtime state: %s", list.Body.String())
		}
		if got := request(); got.Code != http.StatusOK {
			return fmt.Errorf("duplicate queue returned %d: %s", got.Code, got.Body.String())
		} else {
			var payload struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil || payload.Code != 40908 {
				return fmt.Errorf("duplicate queue response: %s error=%v", got.Body.String(), err)
			}
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
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
