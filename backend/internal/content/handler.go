package content

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	repository Repository
	logger     *zap.Logger
}

func NewHandler(repository Repository, logger *zap.Logger) *Handler {
	return &Handler{repository, logger}
}

func (h *Handler) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	if pageSize < 1 {
		pageSize = 12
	}
	if pageSize > 50 {
		pageSize = 50
	}
	articles, total, err := h.repository.ListArticles(c.Request.Context(), c.Query("q"), pageSize, (page-1)*pageSize)
	if err != nil {
		h.logger.Error("list articles", zap.Error(err))
		response.Error(c)
		return
	}
	items := make([]ArticleSummary, 0, len(articles))
	for _, article := range articles {
		items = append(items, article.Summary())
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *Handler) GetArticle(c *gin.Context) {
	article, err := h.repository.GetArticle(c.Request.Context(), c.Param("slug"))
	if errors.Is(err, ErrArticleNotFound) {
		c.JSON(http.StatusNotFound, response.Response{Code: 40401, Message: "article not found", Data: nil})
		return
	}
	if err != nil {
		h.logger.Error("get article", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, article)
}

func (h *Handler) ListLinks(c *gin.Context) {
	links, err := h.repository.ListLinks(c.Request.Context())
	if err != nil {
		h.logger.Error("list links", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, links)
}
