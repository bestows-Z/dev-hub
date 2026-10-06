package media

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const maxCoverBytes = 8 << 20

type Asset struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	OwnerID     uint64    `json:"-"`
	ObjectKey   string    `json:"-"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Asset) TableName() string { return "media_assets" }

func (asset Asset) URL() string { return "/api/v1/media/" + asset.ID }

type Handler struct {
	db     *gorm.DB
	store  *storage.Store
	logger *zap.Logger
}

func NewHandler(db *gorm.DB, store *storage.Store, logger *zap.Logger) *Handler {
	return &Handler{db: db, store: store, logger: logger}
}

func normalizeCover(blob []byte) ([]byte, string, string, error) {
	if len(blob) == 0 || len(blob) > maxCoverBytes {
		return nil, "", "", fmt.Errorf("cover must be at most 8 MiB")
	}
	mime := http.DetectContentType(blob)
	if mime != "image/png" && mime != "image/jpeg" {
		return nil, "", "", fmt.Errorf("cover must be PNG or JPEG")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(blob))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 5000 || config.Height > 5000 || int64(config.Width)*int64(config.Height) > 16_000_000 {
		return nil, "", "", fmt.Errorf("cover dimensions exceed 5000 pixels or 16 megapixels")
	}
	img, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return nil, "", "", fmt.Errorf("cover image cannot be decoded")
	}
	var out bytes.Buffer
	if mime == "image/jpeg" {
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: 85})
		if err == nil && out.Len() <= maxCoverBytes {
			return out.Bytes(), mime, ".jpg", nil
		}
	} else {
		err = png.Encode(&out, img)
		if err == nil && out.Len() <= maxCoverBytes {
			return out.Bytes(), mime, ".png", nil
		}
	}
	if err == nil { err = fmt.Errorf("normalized cover exceeds 8 MiB") }
	return nil, "", "", err
}

func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCoverBytes+(1<<20))
	file, _, err := c.Request.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, "choose a PNG or JPEG cover up to 8 MiB")
		return
	}
	defer file.Close()
	input, err := io.ReadAll(io.LimitReader(file, maxCoverBytes+1))
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, "cannot read cover image")
		return
	}
	body, mime, extension, err := normalizeCover(input)
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, err.Error())
		return
	}
	u := auth.CurrentUser(c)
	if u == nil {
		response.Fail(c, 40102, "authentication required")
		return
	}
	asset := Asset{ID: uuid.NewString(), OwnerID: u.ID, ContentType: mime, SizeBytes: int64(len(body))}
	asset.ObjectKey = "covers/" + asset.ID + extension
	if err := h.store.Put(c.Request.Context(), asset.ObjectKey, body, mime); err != nil {
		h.logger.Error("store cover", zap.Error(err))
		response.Error(c)
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&asset).Error; err != nil {
		_ = h.store.Remove(c.Request.Context(), asset.ObjectKey)
		h.logger.Error("save cover", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"id": asset.ID, "url": asset.URL(), "content_type": mime, "size_bytes": asset.SizeBytes})
}

func (h *Handler) Serve(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	var asset Asset
	if err := h.db.WithContext(c.Request.Context()).First(&asset, "id = ?", id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	object, size, _, err := h.store.Get(c.Request.Context(), asset.ObjectKey)
	if err != nil {
		h.logger.Warn("read cover", zap.Error(err))
		c.Status(http.StatusNotFound)
		return
	}
	defer object.Close()
	c.Header("Content-Type", asset.ContentType)
	c.Header("Content-Length", fmt.Sprint(size))
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, object)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid media id")
		return
	}
	var asset Asset
	if err := h.db.WithContext(c.Request.Context()).First(&asset, "id = ?", id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	u := auth.CurrentUser(c)
	if u == nil || asset.OwnerID != u.ID {
		response.Fail(c, 40301, "media belongs to another user")
		return
	}
	var count int64
	url := asset.URL()
	for _, table := range []string{"articles", "projects", "products"} {
		var used int64
		if err := h.db.WithContext(c.Request.Context()).Table(table).Where("cover_url = ?", url).Count(&used).Error; err != nil {
			response.Error(c)
			return
		}
		count += used
	}
	if count > 0 {
		response.Fail(c, 40909, "cover is in use")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Delete(&asset).Error; err != nil {
		response.Error(c)
		return
	}
	if err := h.store.Remove(c.Request.Context(), asset.ObjectKey); err != nil && !storage.IsNotFound(err) {
		h.logger.Warn("remove cover object", zap.Error(err))
	}
	response.Success(c, gin.H{"deleted": true})
}

func ParseURL(value string) (string, bool) {
	id := strings.TrimPrefix(value, "/api/v1/media/")
	if id == value {
		return "", false
	}
	parsed, err := uuid.Parse(id)
	return id, err == nil && parsed.String() == id
}
