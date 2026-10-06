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
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestAuthorApplicationAndOwnershipIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_POSTGRES") != "1" {
		t.Skip("set DEVHUB_TEST_POSTGRES=1")
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
	rollback := errors.New("rollback author integration")
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		suffix := time.Now().UnixNano() % 1_000_000_000_000
		accounts := make([]user.User, 3)
		for i, role := range []user.Role{user.RoleAdmin, user.RoleUser, user.RoleAuthor} {
			accounts[i] = user.User{Username: fmt.Sprintf("author_test_%d_%d", suffix, i), Email: fmt.Sprintf("author_test_%d_%d@example.invalid", suffix, i), PasswordHash: "test-only", Role: role, Status: user.StatusNormal}
			if err := tx.Create(&accounts[i]).Error; err != nil {
				return err
			}
		}
		ownerID := accounts[2].ID
		article := content.Article{Slug: fmt.Sprintf("owner-test-%d", suffix), Title: "Owner article", BodyMD: "Content", Category: "tech", Tags: []string{}, Status: "draft", AuthorID: &ownerID}
		if err := tx.Create(&article).Error; err != nil {
			return err
		}
		actor := &accounts[1]
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set("currentUser", actor); c.Next() })
		h := NewHandler(tx, nil, zap.NewNop())
		router.POST("/apply", h.ApplyAuthor)
		router.PATCH("/review/:id", h.ReviewAuthorApplication)
		router.POST("/studio", h.CreateMyArticle)
		router.PUT("/studio/:id", h.UpdateMyArticle)
		router.DELETE("/studio/:id", h.DeleteMyArticle)
		request := func(method, path, body string) (int, string) {
			req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var result struct {
				Code int `json:"code"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &result)
			return result.Code, w.Body.String()
		}
		if code, body := request(http.MethodPost, "/apply", `{"reason":"I want to share longer technical articles."}`); code != 0 {
			return fmt.Errorf("apply: %s", body)
		}
		var application AuthorApplication
		if err := tx.Where("user_id = ?", actor.ID).First(&application).Error; err != nil {
			return err
		}
		actor = &accounts[0]
		if code, body := request(http.MethodPatch, fmt.Sprintf("/review/%d", application.ID), `{"status":"approved"}`); code != 0 {
			return fmt.Errorf("review: %s", body)
		}
		if err := tx.First(&accounts[1], accounts[1].ID).Error; err != nil {
			return err
		}
		if accounts[1].Role != user.RoleAuthor {
			return fmt.Errorf("role = %d", accounts[1].Role)
		}
		actor = &accounts[1]
		ownSlug := fmt.Sprintf("own-test-%d", suffix)
		if code, body := request(http.MethodPost, "/studio", fmt.Sprintf(`{"slug":%q,"title":"My article","body_md":"Own content"}`, ownSlug)); code != 0 {
			return fmt.Errorf("own create: %s", body)
		}
		var own content.Article
		if err := tx.Where("slug = ?", ownSlug).First(&own).Error; err != nil {
			return err
		}
		if own.AuthorID == nil || *own.AuthorID != accounts[1].ID {
			return fmt.Errorf("article author not assigned: %+v", own.AuthorID)
		}
		path := fmt.Sprintf("/studio/%d", article.ID)
		payload := fmt.Sprintf(`{"slug":%q,"title":"Hijacked","body_md":"Changed"}`, article.Slug)
		if code, body := request(http.MethodPut, path, payload); code != 40400 {
			return fmt.Errorf("foreign edit code %d: %s", code, body)
		}
		if code, body := request(http.MethodDelete, path, ""); code != 40400 {
			return fmt.Errorf("foreign delete code %d: %s", code, body)
		}
		if err := tx.First(&article, article.ID).Error; err != nil {
			return err
		}
		if article.Title != "Owner article" {
			return fmt.Errorf("foreign article was changed: %q", article.Title)
		}
		if code, body := request(http.MethodPut, fmt.Sprintf("/studio/%d", own.ID), fmt.Sprintf(`{"slug":%q,"title":"My revised article","body_md":"Revised"}`, ownSlug)); code != 0 {
			return fmt.Errorf("own edit: %s", body)
		}
		if err := tx.Model(&article).Update("status", "published").Error; err != nil {
			return err
		}
		public, err := content.NewRepository(tx).GetArticle(t.Context(), article.Slug)
		if err != nil {
			return err
		}
		if public.AuthorName != accounts[2].Username {
			return fmt.Errorf("public author = %q", public.AuthorName)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
