package admin

import (
	"errors"
	"fmt"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/store"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type productInput struct {
	Slug        string `json:"slug" binding:"required,max=160"`
	Name        string `json:"name" binding:"required,max=240"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	PriceCents  int64  `json:"price_cents" binding:"min=0"`
	Stock       int    `json:"stock" binding:"min=0"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
}

func (h *Handler) ListProducts(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&store.Product{})
	if pattern := searchPattern(c); pattern != "" {
		q = q.Where("name ILIKE ? OR slug ILIKE ? OR description ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count admin products", err)
		return
	}
	items := make([]store.Product, 0)
	if err := q.Order("created_at DESC,id DESC").Limit(size).Offset((p - 1) * size).Find(&items).Error; err != nil {
		h.failure(c, "list admin products", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var input productInput
	if err := c.ShouldBindJSON(&input); err != nil || !validSlug(input.Slug) || clean(input.Name) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid product details")
		return
	}
	if !h.requireCover(c, input.CoverURL, "") {
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	item := store.Product{Slug: input.Slug, Name: clean(input.Name), Description: clean(input.Description), CoverURL: clean(input.CoverURL), PriceCents: input.PriceCents, Stock: input.Stock, Status: status}
	if err := h.db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		h.failure(c, "create product", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input productInput
	if err := c.ShouldBindJSON(&input); err != nil || !validSlug(input.Slug) || clean(input.Name) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid product details")
		return
	}
	var item store.Product
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find product", err)
		return
	}
	if !h.requireCover(c, input.CoverURL, item.CoverURL) {
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	item.Slug, item.Name, item.Description, item.CoverURL = input.Slug, clean(input.Name), clean(input.Description), clean(input.CoverURL)
	item.PriceCents, item.Stock, item.Status = input.PriceCents, input.Stock, status
	if err := h.db.WithContext(c.Request.Context()).Save(&item).Error; err != nil {
		h.failure(c, "update product", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var orderCount int64
	if err := h.db.WithContext(c.Request.Context()).Model(&store.Order{}).Where("product_id = ?", id).Count(&orderCount).Error; err != nil {
		h.failure(c, "check product orders", err)
		return
	}
	if orderCount > 0 {
		response.Fail(c, 40905, "product has orders; unpublish it instead")
		return
	}
	result := h.db.WithContext(c.Request.Context()).Delete(&store.Product{}, id)
	if result.Error != nil {
		h.failure(c, "delete product", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40400, "record not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) ListOrders(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&store.Order{}).Joins("LEFT JOIN users ON users.id = orders.user_id")
	if pattern := searchPattern(c); pattern != "" {
		q = q.Where("orders.order_no ILIKE ? OR orders.email ILIKE ? OR users.username ILIKE ?", pattern, pattern, pattern)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("orders.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count orders", err)
		return
	}
	items := make([]orderRow, 0)
	if err := q.Select("orders.*, users.username").Order("orders.created_at DESC,orders.id DESC").Limit(size).Offset((p - 1) * size).Scan(&items).Error; err != nil {
		h.failure(c, "list orders", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

type orderRow struct {
	store.Order
	Username string `json:"username"`
}

type orderStatusInput struct {
	Status string `json:"status" binding:"required,oneof=paid delivered cancelled"`
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input orderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid order status")
		return
	}
	var order store.Order
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return err
		}
		if !allowedTransition(order.Status, input.Status) {
			return errInvalidTransition
		}
		if input.Status == "cancelled" {
			if err := tx.Model(&store.Product{}).Where("id = ?", order.ProductID).Update("stock", gorm.Expr("stock + ?", order.Quantity)).Error; err != nil {
				return fmt.Errorf("restore stock: %w", err)
			}
		}
		order.Status = input.Status
		return tx.Save(&order).Error
	})
	if errors.Is(err, errInvalidTransition) {
		response.Fail(c, 40904, "invalid order status transition")
		return
	}
	if err != nil {
		h.failure(c, "update order status", err)
		return
	}
	response.Success(c, order)
}

var errInvalidTransition = errors.New("invalid order status transition")

func allowedTransition(from, to string) bool {
	return (from == "pending_payment" && (to == "paid" || to == "cancelled")) || (from == "paid" && to == "delivered")
}
