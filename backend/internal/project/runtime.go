package project

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RuntimeHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewRuntimeHandler(db *gorm.DB, logger *zap.Logger) *RuntimeHandler {
	return &RuntimeHandler{db: db, logger: logger}
}

func (h *RuntimeHandler) Serve(c *gin.Context) {
	var item Project
	if err := h.db.WithContext(c.Request.Context()).Where("slug = ? AND status = ? AND runtime_status = ?", c.Param("slug"), "published", "running").First(&item).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	pathPart := strings.TrimPrefix(c.Param("filepath"), "/")
	backend := pathPart == "backend" || strings.HasPrefix(pathPart, "backend/")
	port := item.RuntimeFrontendPort
	if backend {
		port = item.RuntimeBackendPort
		if item.RuntimeBackendPort != item.RuntimeFrontendPort {
			pathPart = strings.TrimPrefix(pathPart, "backend")
			pathPart = strings.TrimPrefix(pathPart, "/")
		}
	}
	if port < 1024 || port > 65535 {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	if c.Request.Method == http.MethodOptions {
		c.Status(http.StatusNoContent)
		return
	}
	target := &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", port)}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(request *http.Request) {
		request.URL.Scheme = target.Scheme
		request.URL.Host = target.Host
		request.URL.Path = "/" + pathPart
		request.URL.RawPath = ""
		request.Host = target.Host
		request.Header.Del("Cookie")
	}
	proxy.ModifyResponse = func(result *http.Response) error {
		result.Header.Del("Set-Cookie")
		result.Header.Set("Access-Control-Allow-Origin", "*")
		result.Header.Set("Cross-Origin-Resource-Policy", "cross-origin")
		result.Header.Set("Referrer-Policy", "no-referrer")
		result.Header.Set("X-Content-Type-Options", "nosniff")
		result.Header.Set("Cache-Control", "no-store")
		result.Header.Set("Content-Security-Policy", "sandbox allow-scripts allow-forms; default-src * data: blob: 'unsafe-inline' 'unsafe-eval'; connect-src *; base-uri 'none'")
		if location := result.Header.Get("Location"); location != "" {
			prefix := fmt.Sprintf("/api/v1/project-runtimes/%s/", item.Slug)
			if backend {
				prefix += "backend/"
			}
			result.Header.Set("Location", rewriteRuntimeLocation(location, prefix, target.Host))
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, request *http.Request, err error) {
		h.logger.Warn("project runtime proxy failed", zap.Error(err))
		http.Error(w, "project runtime unavailable", http.StatusBadGateway)
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func rewriteRuntimeLocation(location, prefix, upstreamHost string) string {
	parsed, err := url.Parse(location)
	if err != nil {
		return location
	}
	if parsed.IsAbs() {
		if parsed.Host != upstreamHost {
			return location
		}
		parsed.Scheme, parsed.Host = "", ""
	}
	if !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(location, "//") {
		return location
	}
	parsed.Path = prefix + strings.TrimPrefix(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed.String()
}
