package admin

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/runtimejobs"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return validSlug(input.Slug) && clean(input.Title) != "" && validPreviewURL && (input.SourceURL == "" || validWebURL(input.SourceURL)) && (input.CoverURL == "" || validWebURL(input.CoverURL) || strings.HasPrefix(input.CoverURL, "/api/v1/media/"))
}

func (h *Handler) ListProjects(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&project.Project{})
	if pattern := searchPattern(c); pattern != "" {
		q = q.Where("title ILIKE ? OR slug ILIKE ? OR description ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count admin projects", err)
		return
	}
	items := make([]projectAdminRow, 0)
	if err := q.Select("projects.*, latest_job.status AS runtime_job_status, latest_job.action AS runtime_job_action, latest_job.error_text AS runtime_job_error").
		Joins("LEFT JOIN LATERAL (SELECT status, action, error_text FROM project_runtime_jobs WHERE project_id = projects.id ORDER BY id DESC LIMIT 1) latest_job ON true").
		Order("projects.created_at DESC,projects.id DESC").Limit(size).Offset((p - 1) * size).Scan(&items).Error; err != nil {
		h.failure(c, "list admin projects", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

type projectAdminRow struct {
	project.Project
	RuntimeJobStatus string `json:"runtime_job_status"`
	RuntimeJobAction string `json:"runtime_job_action"`
	RuntimeJobError  string `json:"runtime_job_error"`
}

func projectHasActiveRuntimeJob(tx *gorm.DB, id uint64) (bool, error) {
	var count int64
	err := tx.Model(&runtimejobs.Job{}).Where("project_id = ? AND status IN ?", id, []string{"queued", "running"}).Count(&count).Error
	return count > 0, err
}

func ensureRuntimeIdle(tx *gorm.DB, id uint64) error {
	active, err := projectHasActiveRuntimeJob(tx, id)
	if err != nil {
		return err
	}
	if active {
		return errRuntimeJobActive
	}
	return nil
}

var errRuntimeJobActive = errors.New("runtime action in progress")
var errRuntimeRunning = errors.New("running project must be stopped first")

func (h *Handler) runtimeMutationFailure(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, errRuntimeJobActive):
		response.Fail(c, 40908, "wait for the runtime action to finish")
	case errors.Is(err, errRuntimeRunning):
		response.Fail(c, 40906, "stop the project runtime before changing its slug or deleting it")
	default:
		h.failure(c, operation, err)
	}
}

func (h *Handler) QueueProjectRuntime(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input struct {
		Action string `json:"action" binding:"required,oneof=start stop"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid runtime action")
		return
	}
	u := auth.CurrentUser(c)
	if u == nil {
		response.Fail(c, 40102, "authentication required")
		return
	}
	var job runtimejobs.Job
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var item project.Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if input.Action == "start" && (item.RuntimeBundleKey == "" || item.RuntimeStatus == "running") {
			return errRuntimeNotReady
		}
		if input.Action == "stop" && item.RuntimeStatus != "running" {
			var last runtimejobs.Job
			if err := tx.Where("project_id = ?", id).Order("id DESC").First(&last).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if last.Status != "failed" {
				return errRuntimeNotReady
			}
		}
		job = runtimejobs.Job{ProjectID: id, Action: input.Action, Status: "queued", RequestedBy: u.ID}
		return tx.Create(&job).Error
	})
	var pgErr *pgconn.PgError
	if errors.Is(err, errRuntimeNotReady) {
		response.Fail(c, 40907, "check the current runtime state and uploaded ZIP")
		return
	}
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		response.Fail(c, 40908, "a runtime action is already in progress")
		return
	}
	if err != nil {
		h.failure(c, "queue project runtime", err)
		return
	}
	response.Success(c, job)
}

var errRuntimeNotReady = errors.New("runtime is not ready for requested action")

func (h *Handler) CreateProject(c *gin.Context) {
	var input projectInput
	if err := c.ShouldBindJSON(&input); err != nil || !validProjectInput(input) {
		response.Fail(c, response.CodeInvalidParams, "invalid project details")
		return
	}
	if !h.requireCover(c, input.CoverURL, "") {
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
	if !h.requireCover(c, input.CoverURL, item.CoverURL) {
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if err := ensureRuntimeIdle(tx, id); err != nil {
			return err
		}
		if item.RuntimeStatus == "running" && item.Slug != input.Slug {
			return errRuntimeRunning
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
		return tx.Save(&item).Error
	})
	if err != nil {
		h.runtimeMutationFailure(c, "update project", err)
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
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if err := ensureRuntimeIdle(tx, id); err != nil {
			return err
		}
		if item.RuntimeStatus == "running" {
			return errRuntimeRunning
		}
		return tx.Delete(&item).Error
	})
	if err != nil {
		h.runtimeMutationFailure(c, "delete project", err)
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
	previous := ""
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if err := ensureRuntimeIdle(tx, id); err != nil {
			return err
		}
		previous = item.RuntimeBundleKey
		return tx.Model(&item).Updates(map[string]any{"runtime_bundle_key": key, "runtime_bundle_uploaded": true}).Error
	})
	if err != nil {
		_ = h.store.Remove(c.Request.Context(), key)
		h.runtimeMutationFailure(c, "save runtime bundle", err)
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
	for _, file := range files {
		if err := h.store.Put(c.Request.Context(), prefix+"/"+file.Path, file.Body, file.ContentType); err != nil {
			_ = h.store.RemovePrefix(c.Request.Context(), prefix)
			h.failure(c, "upload project bundle", err)
			return
		}
	}
	previousPrefix := ""
	previewURL := ""
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if err := ensureRuntimeIdle(tx, id); err != nil {
			return err
		}
		previousPrefix = item.BundlePrefix
		previewURL = fmt.Sprintf("/api/v1/project-previews/%s/index.html", item.Slug)
		updates := map[string]any{"bundle_prefix": prefix}
		if item.RuntimeStatus != "running" {
			updates["preview_url"] = previewURL
		}
		return tx.Model(&item).Updates(updates).Error
	})
	if err != nil {
		_ = h.store.RemovePrefix(c.Request.Context(), prefix)
		h.runtimeMutationFailure(c, "save project bundle", err)
		return
	}
	if previousPrefix != "" {
		if err := h.store.RemovePrefix(c.Request.Context(), previousPrefix); err != nil {
			h.logger.Warn("remove previous project bundle", zap.Error(err))
		}
	}
	response.Success(c, gin.H{"preview_url": previewURL, "files": len(files)})
}
