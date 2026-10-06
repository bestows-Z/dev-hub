package engagement

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

func TestReplyRequiresApprovedCommentOnSameArticle(t *testing.T) {
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
	rollback := errors.New("rollback reply test")
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		suffix := time.Now().UnixNano() % 1_000_000_000_000
		visitor := user.User{Username: fmt.Sprintf("reply_%d", suffix), Email: fmt.Sprintf("reply_%d@example.invalid", suffix), PasswordHash: "test", Role: user.RoleUser, Status: user.StatusNormal}
		if err := tx.Create(&visitor).Error; err != nil {
			return err
		}
		first := content.Article{Slug: fmt.Sprintf("reply-first-%d", suffix), Title: "First", BodyMD: "Content", Category: "tech", Tags: []string{}, Status: "published"}
		second := content.Article{Slug: fmt.Sprintf("reply-second-%d", suffix), Title: "Second", BodyMD: "Content", Category: "tech", Tags: []string{}, Status: "published"}
		if err := tx.Create(&first).Error; err != nil {
			return err
		}
		if err := tx.Create(&second).Error; err != nil {
			return err
		}
		old := time.Now().Add(-time.Hour)
		approved := Comment{ArticleID: first.ID, UserID: visitor.ID, Body: "Original", Status: "approved", CreatedAt: old}
		foreign := Comment{ArticleID: second.ID, UserID: visitor.ID, Body: "Other article", Status: "approved", CreatedAt: old}
		pending := Comment{ArticleID: first.ID, UserID: visitor.ID, Body: "Pending", Status: "pending", CreatedAt: old}
		for _, item := range []*Comment{&approved, &foreign, &pending} {
			if err := tx.Create(item).Error; err != nil {
				return err
			}
		}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set("currentUser", &visitor); c.Next() })
		h := NewHandler(tx, zap.NewNop())
		router.POST("/articles/:slug/comments", h.CreateComment)
		router.GET("/articles/:slug/comments", h.ListComments)
		post := func(id uint64) (int, string) {
			req := httptest.NewRequest(http.MethodPost, "/articles/"+first.Slug+"/comments", bytes.NewBufferString(fmt.Sprintf(`{"body":"My reply","reply_to_id":%d}`, id)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var result struct {
				Code int `json:"code"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &result)
			return result.Code, w.Body.String()
		}
		if code, body := post(foreign.ID); code != 40404 {
			return fmt.Errorf("foreign reply code %d: %s", code, body)
		}
		if code, body := post(pending.ID); code != 40404 {
			return fmt.Errorf("pending reply code %d: %s", code, body)
		}
		if code, body := post(approved.ID); code != 0 {
			return fmt.Errorf("approved reply code %d: %s", code, body)
		}
		var reply Comment
		if err := tx.Where("reply_to_id = ?", approved.ID).Take(&reply).Error; err != nil {
			return err
		}
		if reply.Status != "pending" {
			return fmt.Errorf("reply status = %q", reply.Status)
		}
		if err := tx.Model(&reply).Update("status", "approved").Error; err != nil {
			return err
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/articles/"+first.Slug+"/comments", nil))
		if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"reply_to_username":"`+visitor.Username+`"`)) {
			return fmt.Errorf("reply display: %s", w.Body.String())
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
