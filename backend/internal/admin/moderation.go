package admin

import (
	"errors"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/engagement"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errAlreadyReviewed = errors.New("already reviewed")

type commentReviewRow struct {
	engagement.Comment
	Username     string `json:"username"`
	ArticleTitle string `json:"article_title"`
}

func (h *Handler) ListComments(c *gin.Context) {
	p, size := page(c)
	status := c.DefaultQuery("status", "pending")
	if status != "pending" && status != "approved" && status != "rejected" {
		response.Fail(c, response.CodeInvalidParams, "invalid comment status")
		return
	}
	q := h.db.WithContext(c.Request.Context()).Model(&engagement.Comment{}).Where("article_comments.status = ?", status)
	if pattern := searchPattern(c); pattern != "" {
		q = q.Joins("JOIN users ON users.id = article_comments.user_id").Joins("JOIN articles ON articles.id = article_comments.article_id").Where("article_comments.body ILIKE ? OR users.username ILIKE ? OR articles.title ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count comments for review", err)
		return
	}
	items := make([]commentReviewRow, 0)
	listQuery := q.Select("article_comments.*, users.username, articles.title AS article_title")
	if searchPattern(c) == "" {
		listQuery = listQuery.Joins("JOIN users ON users.id = article_comments.user_id").Joins("JOIN articles ON articles.id = article_comments.article_id")
	}
	if err := listQuery.Order("article_comments.created_at ASC, article_comments.id ASC").Limit(size).Offset((p - 1) * size).Scan(&items).Error; err != nil {
		h.failure(c, "list comments for review", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) ReviewComment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status" binding:"required,oneof=approved rejected"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid review status")
		return
	}
	result := h.db.WithContext(c.Request.Context()).Model(&engagement.Comment{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{"status": input.Status, "reviewed_at": time.Now()})
	if result.Error != nil {
		h.failure(c, "review comment", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40902, "comment is no longer pending")
		return
	}
	response.Success(c, gin.H{"id": id, "status": input.Status})
}

type linkApplicationReviewRow struct {
	engagement.LinkApplication
	Username string `json:"username"`
}

func (h *Handler) ListLinkApplications(c *gin.Context) {
	p, size := page(c)
	status := c.DefaultQuery("status", "pending")
	if status != "pending" && status != "approved" && status != "rejected" {
		response.Fail(c, response.CodeInvalidParams, "invalid application status")
		return
	}
	q := h.db.WithContext(c.Request.Context()).Model(&engagement.LinkApplication{}).Where("link_applications.status = ?", status)
	if pattern := searchPattern(c); pattern != "" {
		q = q.Joins("JOIN users ON users.id = link_applications.user_id").Where("link_applications.name ILIKE ? OR link_applications.url ILIKE ? OR users.username ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count link applications", err)
		return
	}
	items := make([]linkApplicationReviewRow, 0)
	listQuery := q.Select("link_applications.*, users.username")
	if searchPattern(c) == "" {
		listQuery = listQuery.Joins("JOIN users ON users.id = link_applications.user_id")
	}
	if err := listQuery.Order("link_applications.created_at ASC, link_applications.id ASC").Limit(size).Offset((p - 1) * size).Scan(&items).Error; err != nil {
		h.failure(c, "list link applications", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) ReviewLinkApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status" binding:"required,oneof=approved rejected"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid review status")
		return
	}
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var item engagement.LinkApplication
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if item.Status != "pending" {
			return errAlreadyReviewed
		}
		if input.Status == "approved" {
			link := content.FriendLink{Name: item.Name, URL: item.URL, AvatarURL: item.AvatarURL, Description: item.Description, Enabled: true}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return tx.Model(&item).Updates(map[string]any{"status": input.Status, "reviewed_at": time.Now()}).Error
	})
	if errors.Is(err, errAlreadyReviewed) {
		response.Fail(c, 40902, "application is no longer pending")
		return
	}
	if err != nil {
		h.failure(c, "review link application", err)
		return
	}
	response.Success(c, gin.H{"id": id, "status": input.Status})
}
