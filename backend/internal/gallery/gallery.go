package gallery

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Item struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	ObjectKey   string     `json:"-"`
	ImageURL    string     `json:"image_url" gorm:"-"`
	Status      string     `json:"status"`
	TakenAt     *time.Time `json:"taken_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
}

func (Item) TableName() string { return "gallery_items" }

func (item *Item) SetImageURL() {
	if item.ObjectKey != "" {
		item.ImageURL = "/api/v1/gallery/" + strconv.FormatUint(item.ID, 10) + "/image?v=" + strconv.FormatInt(item.UpdatedAt.UnixNano(), 10)
	}
}

type Handler struct {
	db     *gorm.DB
	store  *storage.Store
	logger *zap.Logger
}

func NewHandler(db *gorm.DB, store *storage.Store, logger *zap.Logger) *Handler {
	return &Handler{db: db, store: store, logger: logger}
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
	if size > 36 {
		size = 36
	}
	query := h.db.WithContext(c.Request.Context()).Model(&Item{}).Where("status = ? AND object_key <> ''", "published")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("count gallery", zap.Error(err))
		response.Error(c)
		return
	}
	items := make([]Item, 0)
	if err := query.Order("created_at DESC, id DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		h.logger.Error("list gallery", zap.Error(err))
		response.Error(c)
		return
	}
	for i := range items {
		items[i].SetImageURL()
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}

func (h *Handler) Image(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	var item Item
	if err := h.db.WithContext(c.Request.Context()).Where("id = ? AND status = ? AND object_key <> ''", id, "published").Take(&item).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	object, size, contentType, err := h.store.Get(c.Request.Context(), item.ObjectKey)
	if err != nil {
		h.logger.Warn("read gallery image", zap.Error(err))
		c.Status(http.StatusNotFound)
		return
	}
	defer object.Close()
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, object)
}
