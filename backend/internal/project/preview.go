package project

import (
	"net/http"
	"path"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PreviewHandler struct {
	db     *gorm.DB
	store  *storage.Store
	logger *zap.Logger
}

func NewPreviewHandler(db *gorm.DB, store *storage.Store, logger *zap.Logger) *PreviewHandler {
	return &PreviewHandler{db: db, store: store, logger: logger}
}

func (h *PreviewHandler) Serve(c *gin.Context) {
	var item Project
	if err := h.db.WithContext(c.Request.Context()).Where("slug = ? AND status = ? AND bundle_prefix <> ''", c.Param("slug"), "published").First(&item).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	file := strings.TrimPrefix(c.Param("filepath"), "/")
	if file == "" {
		file = "index.html"
	}
	if path.Clean(file) != file || strings.HasPrefix(file, "../") || strings.Contains(file, `\`) {
		c.Status(http.StatusBadRequest)
		return
	}
	contentType, ok := contentTypes[strings.ToLower(path.Ext(file))]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	object, size, _, err := h.store.Get(c.Request.Context(), item.BundlePrefix+"/"+file)
	if err != nil {
		if !storage.IsNotFound(err) {
			h.logger.Warn("load project preview", zap.Error(err))
		}
		c.Status(http.StatusNotFound)
		return
	}
	defer object.Close()
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Cross-Origin-Resource-Policy": "cross-origin",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"Cache-Control":                "public, max-age=60",
	}
	if strings.EqualFold(path.Ext(file), ".html") || strings.EqualFold(path.Ext(file), ".htm") {
		headers["Content-Security-Policy"] = "sandbox allow-scripts; default-src * data: blob: 'unsafe-inline' 'unsafe-eval'; connect-src 'none'; form-action 'none'; base-uri 'none'"
	}
	c.DataFromReader(http.StatusOK, size, contentType, object, headers)
}
