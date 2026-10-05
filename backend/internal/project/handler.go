package project

import (
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

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	if size < 1 {
		size = 12
	}
	if size > 50 {
		size = 50
	}
	projects, total, err := h.repository.List(c.Request.Context(), size, (page-1)*size)
	if err != nil {
		h.logger.Error("list projects", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"items": projects, "total": total, "page": page, "page_size": size})
}
