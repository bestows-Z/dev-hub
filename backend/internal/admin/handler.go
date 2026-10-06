package admin

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/media"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	logger *zap.Logger
	store  *storage.Store
	index  ArticleIndex
}

type ArticleIndex interface {
	SyncArticle(context.Context, content.Article) error
	DeleteArticle(context.Context, uint64) error
	Disable()
}

func NewHandler(db *gorm.DB, store *storage.Store, logger *zap.Logger) *Handler {
	return &Handler{db: db, store: store, logger: logger}
}

func (h *Handler) SetArticleIndex(index ArticleIndex) { h.index = index }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validSlug(value string) bool { return len(value) <= 160 && slugPattern.MatchString(value) }

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeInvalidParams, "invalid id")
		return 0, false
	}
	return id, true
}

func (h *Handler) failure(c *gin.Context, operation string, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, response.Response{Code: 40400, Message: "record not found"})
		return
	}
	h.logger.Error(operation, zap.Error(err))
	response.Error(c)
}

func page(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	return page, size
}

func clean(value string) string { return strings.TrimSpace(value) }

func (h *Handler) requireCover(c *gin.Context, url, previous string) bool {
	url = clean(url)
	if url == "" || (previous != "" && url == previous) {
		return true
	}
	id, ok := media.ParseURL(url)
	if !ok {
		response.Fail(c, response.CodeInvalidParams, "upload a cover image instead of entering an external URL")
		return false
	}
	var count int64
	query := h.db.WithContext(c.Request.Context()).Model(&media.Asset{}).Where("id = ?", id)
	if current := auth.CurrentUser(c); current != nil && current.Role != user.RoleAdmin {
		query = query.Where("owner_id = ?", current.ID)
	}
	if err := query.Count(&count).Error; err != nil {
		h.failure(c, "check cover image", err)
		return false
	}
	if count == 0 {
		response.Fail(c, response.CodeInvalidParams, "cover image does not exist")
		return false
	}
	return true
}
