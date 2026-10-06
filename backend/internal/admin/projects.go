package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	localPreview := strings.TrimPrefix(input.PreviewURL, "/api/v1/project-previews/")
	localSlug := strings.TrimSuffix(localPreview, "/index.html")
	runtimePreview := strings.TrimPrefix(input.PreviewURL, "/api/v1/project-runtimes/")
	runtimeSlug := strings.TrimSuffix(runtimePreview, "/")
	validPreviewURL := input.PreviewURL == "" || validWebURL(input.PreviewURL) || (localPreview != input.PreviewURL && validSlug(localSlug) && localPreview == localSlug+"/index.html") || (runtimePreview != input.PreviewURL && validSlug(runtimeSlug) && runtimePreview == runtimeSlug+"/")
	return validSlug(input.Slug) && clean(input.Title) != "" && validPreviewURL && (input.SourceURL == "" || validWebURL(input.SourceURL)) && (input.CoverURL == "" || validWebURL(input.CoverURL))
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
	item := project.Project{Slug: input.Slug, Title: clean(input.Title), Description: clean(input.Description), CoverURL: input.CoverURL, Tags: input.Tags, PreviewURL: input.PreviewURL, RuntimeStatus: "stopped", SourceURL: input.SourceURL, Status: status}
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
	if item.RuntimeStatus == "running" {
		item.PreviewURL = fmt.Sprintf("/api/v1/project-runtimes/%s/", item.Slug)
		item.BackendURL = fmt.Sprintf("/api/v1/project-runtimes/%s/backend/", item.Slug)
	} else if item.BundlePrefix != "" {
		item.PreviewURL = fmt.Sprintf("/api/v1/project-previews/%s/index.html", item.Slug)
	}
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
	var item project.Project
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find project", err)
		return
	}
	if item.RuntimeStatus == "running" {
		response.Fail(c, 40906, "stop the project runtime before deleting it")
		return
	}
	result := h.db.WithContext(c.Request.Context()).Delete(&item)
	if result.Error != nil {
		h.failure(c, "delete project", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, 40400, "record not found")
		return
	}
	if item.BundlePrefix != "" {
		if err := h.store.RemovePrefix(c.Request.Context(), item.BundlePrefix); err != nil {
			h.logger.Warn("remove project bundle after delete", zap.Error(err))
		}
	}
	if item.RuntimeBundleKey != "" {
		if err := h.store.Remove(c.Request.Context(), item.RuntimeBundleKey); err != nil {
			h.logger.Warn("remove runtime bundle after delete", zap.Error(err))
		}
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) UploadRuntimeBundle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item project.Project
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find project for runtime upload", err)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, project.MaxRuntimeBundleBytes+(1<<20))
	file, err := c.FormFile("file")
	if err != nil || file.Size == 0 || file.Size > project.MaxRuntimeBundleBytes || !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		response.Fail(c, response.CodeInvalidParams, "upload a runtime ZIP of at most 50 MiB")
		return
	}
	handle, err := file.Open()
	if err != nil {
		h.failure(c, "open runtime bundle", err)
		return
	}
	defer handle.Close()
	blob, err := io.ReadAll(io.LimitReader(handle, project.MaxRuntimeBundleBytes+1))
	if err != nil || len(blob) > project.MaxRuntimeBundleBytes {
		response.Fail(c, response.CodeInvalidParams, "runtime ZIP exceeds 50 MiB")
		return
	}
	if err := project.ValidateRuntimeBundle(blob); err != nil {
		response.Fail(c, response.CodeInvalidParams, err.Error())
		return
	}
	if err := h.store.EnsureBucket(c.Request.Context()); err != nil {
		h.failure(c, "prepare runtime bucket", err)
		return
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		h.failure(c, "generate runtime revision", err)
		return
	}
	key := fmt.Sprintf("project-runtimes/%d/%s.zip", item.ID, hex.EncodeToString(random[:]))
	if err := h.store.Put(c.Request.Context(), key, blob, "application/zip"); err != nil {
		h.failure(c, "upload runtime bundle", err)
		return
	}
	previous := item.RuntimeBundleKey
	if err := h.db.WithContext(c.Request.Context()).Model(&item).Updates(map[string]any{"runtime_bundle_key": key, "runtime_bundle_uploaded": true}).Error; err != nil {
		_ = h.store.Remove(c.Request.Context(), key)
		h.failure(c, "save runtime bundle", err)
		return
	}
	if previous != "" {
		if err := h.store.Remove(c.Request.Context(), previous); err != nil {
			h.logger.Warn("remove previous runtime bundle", zap.Error(err))
		}
	}
	response.Success(c, gin.H{"uploaded": true, "runtime_status": item.RuntimeStatus})
}

func (h *Handler) UploadProjectBundle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item project.Project
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find project for upload", err)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, project.MaxBundleBytes+(1<<20))
	file, err := c.FormFile("file")
	if err != nil || file.Size == 0 || file.Size > project.MaxBundleBytes || !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		response.Fail(c, response.CodeInvalidParams, "upload a ZIP of at most 20 MiB")
		return
	}
	handle, err := file.Open()
	if err != nil {
		h.failure(c, "open project bundle", err)
		return
	}
	defer handle.Close()
	blob, err := io.ReadAll(io.LimitReader(handle, project.MaxBundleBytes+1))
	if err != nil || len(blob) > project.MaxBundleBytes {
		response.Fail(c, response.CodeInvalidParams, "ZIP exceeds 20 MiB")
		return
	}
	files, err := project.ParseStaticBundle(blob)
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, err.Error())
		return
	}
	if err := h.store.EnsureBucket(c.Request.Context()); err != nil {
		h.failure(c, "prepare project bucket", err)
		return
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		h.failure(c, "generate bundle revision", err)
		return
	}
	prefix := fmt.Sprintf("projects/%d/%s", item.ID, hex.EncodeToString(random[:]))
	previousPrefix := item.BundlePrefix
	for _, file := range files {
		if err := h.store.Put(c.Request.Context(), prefix+"/"+file.Path, file.Body, file.ContentType); err != nil {
			_ = h.store.RemovePrefix(c.Request.Context(), prefix)
			h.failure(c, "upload project bundle", err)
			return
		}
	}
	previewURL := fmt.Sprintf("/api/v1/project-previews/%s/index.html", item.Slug)
	updates := map[string]any{"bundle_prefix": prefix}
	if item.RuntimeStatus != "running" {
		updates["preview_url"] = previewURL
	}
	if err := h.db.WithContext(c.Request.Context()).Model(&item).Updates(updates).Error; err != nil {
		_ = h.store.RemovePrefix(c.Request.Context(), prefix)
		h.failure(c, "save project bundle", err)
		return
	}
	if previousPrefix != "" {
		if err := h.store.RemovePrefix(c.Request.Context(), previousPrefix); err != nil {
			h.logger.Warn("remove previous project bundle", zap.Error(err))
		}
	}
	response.Success(c, gin.H{"preview_url": previewURL, "files": len(files)})
}
