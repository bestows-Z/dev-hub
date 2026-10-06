package analytics

import (
	"net/http"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	store  *Store
	logger *zap.Logger
}

func NewHandler(store *Store, logger *zap.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) Summary(c *gin.Context) {
	if h == nil || h.store == nil {
		response.Fail(c, 50301, "visit analytics is unavailable")
		return
	}
	summary, err := h.store.Summary(c.Request.Context())
	if err != nil {
		h.logger.Error("load visit analytics", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, summary)
}

func Tracker(broker *Broker) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if broker == nil || c.Request.Method != http.MethodGet || c.Writer.Status() != http.StatusOK {
			return
		}
		var kind string
		switch c.FullPath() {
		case "/api/v1/articles/:slug":
			kind = "article"
		case "/api/v1/products/:slug":
			kind = "product"
		case "/api/v1/project-previews/:slug/*filepath", "/api/v1/project-runtimes/:slug/*filepath":
			if c.Param("filepath") != "/" && c.Param("filepath") != "/index.html" {
				return
			}
			kind = "project"
		default:
			return
		}
		if slug := c.Param("slug"); slug != "" {
			broker.Record(kind, slug)
		}
	}
}
