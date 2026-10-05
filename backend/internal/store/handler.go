package store

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

func (h *Handler) ListProducts(c *gin.Context) {
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
	products, total, err := h.repository.ListProducts(c.Request.Context(), size, (page-1)*size)
	if err != nil {
		h.logger.Error("list products", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"items": products, "total": total, "page": page, "page_size": size})
}

func (h *Handler) GetProduct(c *gin.Context) {
	product, err := h.repository.GetProduct(c.Request.Context(), c.Param("slug"))
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, response.Response{Code: 40402, Message: "product not found", Data: nil})
		return
	}
	if err != nil {
		h.logger.Error("get product", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, product)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid order details")
		return
	}
	order, err := h.repository.CreateOrder(c.Request.Context(), req)
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, 40402, "product not found")
	case errors.Is(err, ErrOutOfStock):
		response.Fail(c, 40903, "product out of stock")
	case err != nil:
		h.logger.Error("create order", zap.Error(err))
		response.Error(c)
	default:
		response.SuccessMessage(c, "order created", order)
	}
}
