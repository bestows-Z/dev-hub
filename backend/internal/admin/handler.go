package admin

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewHandler(db *gorm.DB, logger *zap.Logger) *Handler { return &Handler{db, logger} }

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
