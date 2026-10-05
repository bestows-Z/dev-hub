package admin

import (
	"net/url"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
)

type articleInput struct {
	Slug     string   `json:"slug" binding:"required,max=160"`
	Title    string   `json:"title" binding:"required,max=240"`
	Excerpt  string   `json:"excerpt"`
	BodyMD   string   `json:"body_md" binding:"required"`
	CoverURL string   `json:"cover_url"`
	Category string   `json:"category" binding:"omitempty,oneof=tech travel essay record"`
	Tags     []string `json:"tags"`
	Status   string   `json:"status" binding:"omitempty,oneof=draft published"`
}

func (h *Handler) ListArticles(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&content.Article{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count admin articles", err)
		return
	}
	items := make([]content.Article, 0)
	if err := q.Order("created_at DESC, id DESC").Limit(size).Offset((p - 1) * size).Find(&items).Error; err != nil {
		h.failure(c, "list admin articles", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) CreateArticle(c *gin.Context) {
	var input articleInput
	if err := c.ShouldBindJSON(&input); err != nil || !validSlug(input.Slug) || clean(input.Title) == "" || clean(input.BodyMD) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid article details")
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	category := input.Category
	if category == "" {
		category = "tech"
	}
	article := content.Article{Slug: input.Slug, Title: clean(input.Title), Excerpt: clean(input.Excerpt), BodyMD: input.BodyMD, CoverURL: clean(input.CoverURL), Category: category, Tags: input.Tags, Status: status}
	if article.Tags == nil {
		article.Tags = []string{}
	}
	if status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&article).Error; err != nil {
		h.failure(c, "create article", err)
		return
	}
	response.Success(c, article)
}

func (h *Handler) UpdateArticle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input articleInput
	if err := c.ShouldBindJSON(&input); err != nil || !validSlug(input.Slug) || clean(input.Title) == "" || clean(input.BodyMD) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid article details")
		return
	}
	var article content.Article
	if err := h.db.WithContext(c.Request.Context()).First(&article, id).Error; err != nil {
		h.failure(c, "find article", err)
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	article.Slug, article.Title, article.Excerpt, article.BodyMD = input.Slug, clean(input.Title), clean(input.Excerpt), input.BodyMD
	article.CoverURL, article.Tags, article.Status = clean(input.CoverURL), input.Tags, status
	if input.Category != "" {
		article.Category = input.Category
	}
	if article.Tags == nil {
		article.Tags = []string{}
	}
	if status == "published" && article.PublishedAt == nil {
		now := time.Now()
		article.PublishedAt = &now
	}
	if status == "draft" {
		article.PublishedAt = nil
	}
	if err := h.db.WithContext(c.Request.Context()).Save(&article).Error; err != nil {
		h.failure(c, "update article", err)
		return
	}
	response.Success(c, article)
}

func (h *Handler) DeleteArticle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result := h.db.WithContext(c.Request.Context()).Delete(&content.Article{}, id)
	if result.Error != nil {
		h.failure(c, "delete article", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40400, "record not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type linkInput struct {
	Name        string `json:"name" binding:"required,max=100"`
	URL         string `json:"url" binding:"required,url"`
	AvatarURL   string `json:"avatar_url"`
	Description string `json:"description" binding:"max=280"`
	SortOrder   int    `json:"sort_order"`
	Enabled     *bool  `json:"enabled"`
}

func validWebURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "https" || parsed.Scheme == "http")
}

func (h *Handler) ListLinks(c *gin.Context) {
	items := make([]content.FriendLink, 0)
	if err := h.db.WithContext(c.Request.Context()).Order("sort_order ASC,id ASC").Find(&items).Error; err != nil {
		h.failure(c, "list admin links", err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateLink(c *gin.Context) {
	var input linkInput
	if err := c.ShouldBindJSON(&input); err != nil || clean(input.Name) == "" || !validWebURL(input.URL) || (input.AvatarURL != "" && !validWebURL(input.AvatarURL)) {
		response.Fail(c, response.CodeInvalidParams, "invalid link details")
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	item := content.FriendLink{Name: clean(input.Name), URL: input.URL, AvatarURL: input.AvatarURL, Description: clean(input.Description), SortOrder: input.SortOrder, Enabled: enabled}
	if err := h.db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		h.failure(c, "create link", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input linkInput
	if err := c.ShouldBindJSON(&input); err != nil || clean(input.Name) == "" || !validWebURL(input.URL) || (input.AvatarURL != "" && !validWebURL(input.AvatarURL)) {
		response.Fail(c, response.CodeInvalidParams, "invalid link details")
		return
	}
	var item content.FriendLink
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find link", err)
		return
	}
	item.Name, item.URL, item.AvatarURL, item.Description, item.SortOrder = clean(input.Name), input.URL, input.AvatarURL, clean(input.Description), input.SortOrder
	if input.Enabled != nil {
		item.Enabled = *input.Enabled
	}
	if err := h.db.WithContext(c.Request.Context()).Save(&item).Error; err != nil {
		h.failure(c, "update link", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) DeleteLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result := h.db.WithContext(c.Request.Context()).Delete(&content.FriendLink{}, id)
	if result.Error != nil {
		h.failure(c, "delete link", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40400, "record not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
