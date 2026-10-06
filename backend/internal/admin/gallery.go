package admin

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/gallery"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type galleryInput struct {
	Title       string `json:"title" binding:"required,max=180"`
	Description string `json:"description" binding:"max=500"`
	Location    string `json:"location" binding:"max=120"`
	TakenAt     string `json:"taken_at"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
}

func galleryDate(value string) (*time.Time, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	return &parsed, err == nil
}

func (h *Handler) ListGallery(c *gin.Context) {
	p, size := page(c)
	query := h.db.WithContext(c.Request.Context()).Model(&gallery.Item{})
	if pattern := searchPattern(c); pattern != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ? OR location ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.failure(c, "count admin gallery", err)
		return
	}
	items := make([]gallery.Item, 0)
	if err := query.Order("created_at DESC, id DESC").Limit(size).Offset((p - 1) * size).Find(&items).Error; err != nil {
		h.failure(c, "list admin gallery", err)
		return
	}
	for i := range items {
		items[i].SetImageURL()
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

func (h *Handler) CreateGallery(c *gin.Context) {
	var input galleryInput
	if err := c.ShouldBindJSON(&input); err != nil || clean(input.Title) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid gallery details")
		return
	}
	takenAt, valid := galleryDate(strings.TrimSpace(input.TakenAt))
	if !valid || input.Status == "published" {
		response.Fail(c, response.CodeInvalidParams, "upload an image before publishing")
		return
	}
	item := gallery.Item{Title: clean(input.Title), Description: clean(input.Description), Location: clean(input.Location), TakenAt: takenAt, Status: "draft"}
	if err := h.db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		h.failure(c, "create gallery item", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdateGallery(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input galleryInput
	if err := c.ShouldBindJSON(&input); err != nil || clean(input.Title) == "" {
		response.Fail(c, response.CodeInvalidParams, "invalid gallery details")
		return
	}
	takenAt, valid := galleryDate(strings.TrimSpace(input.TakenAt))
	if !valid {
		response.Fail(c, response.CodeInvalidParams, "invalid photo date")
		return
	}
	var item gallery.Item
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find gallery item", err)
		return
	}
	status := input.Status
	if status == "" {
		status = "draft"
	}
	if status == "published" && item.ObjectKey == "" {
		response.Fail(c, response.CodeInvalidParams, "upload an image before publishing")
		return
	}
	item.Title, item.Description, item.Location, item.TakenAt, item.Status = clean(input.Title), clean(input.Description), clean(input.Location), takenAt, status
	if err := h.db.WithContext(c.Request.Context()).Save(&item).Error; err != nil {
		h.failure(c, "update gallery item", err)
		return
	}
	item.SetImageURL()
	response.Success(c, item)
}

func (h *Handler) UploadGalleryImage(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item gallery.Item
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find gallery item", err)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 12<<20)
	file, err := c.FormFile("file")
	if err != nil || file.Size == 0 || file.Size > 10<<20 {
		response.Fail(c, response.CodeInvalidParams, "image must be 10 MiB or smaller")
		return
	}
	reader, err := file.Open()
	if err != nil {
		h.failure(c, "open gallery upload", err)
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 10<<20+1))
	if err != nil || len(data) == 0 || len(data) > 10<<20 {
		response.Fail(c, response.CodeInvalidParams, "image must be 10 MiB or smaller")
		return
	}
	contentType := http.DetectContentType(data)
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}[contentType]
	if extension == "" {
		response.Fail(c, response.CodeInvalidParams, "only JPEG, PNG, WebP and GIF images are supported")
		return
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		h.failure(c, "generate gallery key", err)
		return
	}
	key := "gallery/" + strconv.FormatUint(id, 10) + "/" + hex.EncodeToString(nonce) + extension
	if err := h.store.Put(c.Request.Context(), key, data, contentType); err != nil {
		h.failure(c, "upload gallery image", err)
		return
	}
	previousKey := item.ObjectKey
	item.ObjectKey = key
	if err := h.db.WithContext(c.Request.Context()).Save(&item).Error; err != nil {
		_ = h.store.Remove(c.Request.Context(), key)
		h.failure(c, "save gallery image", err)
		return
	}
	if previousKey != "" {
		if err := h.store.Remove(c.Request.Context(), previousKey); err != nil {
			h.logger.Warn("remove replaced gallery image", zap.Error(err))
		}
	}
	item.SetImageURL()
	response.Success(c, item)
}

func (h *Handler) DeleteGallery(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item gallery.Item
	if err := h.db.WithContext(c.Request.Context()).First(&item, id).Error; err != nil {
		h.failure(c, "find gallery item", err)
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Delete(&item).Error; err != nil {
		h.failure(c, "delete gallery item", err)
		return
	}
	if item.ObjectKey != "" {
		if err := h.store.Remove(c.Request.Context(), item.ObjectKey); err != nil {
			h.logger.Warn("remove deleted gallery image", zap.Error(err))
		}
	}
	response.Success(c, gin.H{"deleted": true})
}
