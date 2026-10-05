package admin

import (
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/gin-gonic/gin"
)

type projectInput struct {
	Slug        string   `json:"slug" binding:"required,max=160"`
	Title       string   `json:"title" binding:"required,max=240"`
	Description string   `json:"description"`
	CoverURL    string   `json:"cover_url"`
	Tags        []string `json:"tags"`
	PreviewURL  string   `json:"preview_url"`
	SourceURL   string   `json:"source_url"`
	Status      string   `json:"status" binding:"omitempty,oneof=draft published"`
}

func validProjectInput(input projectInput) bool {
	return validSlug(input.Slug) && clean(input.Title) != "" && (input.PreviewURL == "" || validWebURL(input.PreviewURL)) && (input.SourceURL == "" || validWebURL(input.SourceURL)) && (input.CoverURL == "" || validWebURL(input.CoverURL))
}

func (h *Handler) ListProjects(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&project.Project{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count admin projects", err)
		return
	}
	items := make([]project.Project, 0)
	if err := q.Order("created_at DESC,id DESC").Limit(size).Offset((p - 1) * size).Find(&items).Error; err != nil {
		h.failure(c, "list admin projects", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) CreateProject(c *gin.Context) {
	var input projectInput
	if err := c.ShouldBindJSON(&input); err != nil || !validProjectInput(input) {
		response.Fail(c, response.CodeInvalidParams, "invalid project details")
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	item := project.Project{Slug: input.Slug, Title: clean(input.Title), Description: clean(input.Description), CoverURL: input.CoverURL, Tags: input.Tags, PreviewURL: input.PreviewURL, SourceURL: input.SourceURL, Status: status}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		h.failure(c, "create project", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input projectInput
	if err := c.ShouldBindJSON(&input); err != nil || !validProjectInput(input) {
		response.Fail(c, response.CodeInvalidParams, "invalid project details")
		return
	}
	var item project.Project
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find project", err)
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	item.Slug, item.Title, item.Description, item.CoverURL = input.Slug, clean(input.Title), clean(input.Description), input.CoverURL
	item.Tags, item.PreviewURL, item.SourceURL, item.Status = input.Tags, input.PreviewURL, input.SourceURL, status
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if err := h.db.WithContext(c.Request.Context()).Save(&item).Error; err != nil {
		h.failure(c, "update project", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result := h.db.WithContext(c.Request.Context()).Delete(&project.Project{}, id)
	if result.Error != nil {
		h.failure(c, "delete project", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40400, "record not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
