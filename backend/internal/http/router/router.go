package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/admin"
	"github.com/bestows-Z/dev-hub/backend/internal/analytics"
	"github.com/bestows-Z/dev-hub/backend/internal/assistant"
	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/engagement"
	"github.com/bestows-Z/dev-hub/backend/internal/gallery"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/store"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
)

func New(sqlDB *sql.DB, userHandler *user.Handler, authHandler *auth.Handler, contentHandler *content.Handler, storeHandler *store.Handler, projectHandler *project.Handler, previewHandler *project.PreviewHandler, runtimeHandler *project.RuntimeHandler, galleryHandler *gallery.Handler, adminHandler *admin.Handler, assistantHandler *assistant.Handler, engagementHandler *engagement.Handler, analyticsHandler *analytics.Handler, analyticsBroker *analytics.Broker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(analytics.Tracker(analyticsBroker))
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"service": "devhub-api",
				"status":  "up",
			},
		})
	})
	v1.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			time.Second,
		)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			response.Error(c)
			return
		}

		response.Success(
			c,
			gin.H{
				"postgres": "up",
			},
		)
	})
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}
	v1.GET("/auth/me", authHandler.RequireUser(), authHandler.Me)
	v1.PUT("/auth/me", authHandler.RequireUser(), authHandler.UpdateMe)
	v1.POST("/auth/me/avatar", authHandler.RequireUser(), authHandler.UploadAvatar)
	v1.DELETE("/auth/me/avatar", authHandler.RequireUser(), authHandler.DeleteAvatar)
	v1.GET("/avatars/:id", authHandler.ServeAvatar)
	v1.GET("/articles", contentHandler.ListArticles)
	v1.GET("/articles/facets", contentHandler.ArticleFacets)
	v1.GET("/articles/:slug", contentHandler.GetArticle)
	v1.GET("/articles/:slug/comments", engagementHandler.ListComments)
	v1.POST("/articles/:slug/comments", authHandler.RequireUser(), engagementHandler.CreateComment)
	v1.GET("/links", contentHandler.ListLinks)
	v1.GET("/gallery", galleryHandler.List)
	v1.GET("/gallery/:id/image", galleryHandler.Image)
	v1.POST("/link-applications", authHandler.RequireUser(), engagementHandler.ApplyLink)
	v1.GET("/products", storeHandler.ListProducts)
	v1.GET("/products/:slug", storeHandler.GetProduct)
	v1.POST("/orders", storeHandler.CreateOrder)
	v1.GET("/projects", projectHandler.List)
	v1.GET("/project-previews/:slug/*filepath", previewHandler.Serve)
	v1.Any("/project-runtimes/:slug/*filepath", runtimeHandler.Serve)
	v1.POST("/assistant/chat", assistantHandler.Chat)
	adminGroup := v1.Group("/admin", authHandler.RequireUser(), authHandler.RequireAdmin())
	adminGroup.GET("/articles", adminHandler.ListArticles)
	adminGroup.GET("/gallery", adminHandler.ListGallery)
	adminGroup.POST("/gallery", adminHandler.CreateGallery)
	adminGroup.PUT("/gallery/:id", adminHandler.UpdateGallery)
	adminGroup.POST("/gallery/:id/image", adminHandler.UploadGalleryImage)
	adminGroup.DELETE("/gallery/:id", adminHandler.DeleteGallery)
	adminGroup.POST("/articles", adminHandler.CreateArticle)
	adminGroup.PUT("/articles/:id", adminHandler.UpdateArticle)
	adminGroup.DELETE("/articles/:id", adminHandler.DeleteArticle)
	adminGroup.GET("/links", adminHandler.ListLinks)
	adminGroup.POST("/links", adminHandler.CreateLink)
	adminGroup.PUT("/links/:id", adminHandler.UpdateLink)
	adminGroup.DELETE("/links/:id", adminHandler.DeleteLink)
	adminGroup.GET("/comments", adminHandler.ListComments)
	adminGroup.PATCH("/comments/:id", adminHandler.ReviewComment)
	adminGroup.GET("/link-applications", adminHandler.ListLinkApplications)
	adminGroup.PATCH("/link-applications/:id", adminHandler.ReviewLinkApplication)
	adminGroup.GET("/products", adminHandler.ListProducts)
	adminGroup.POST("/products", adminHandler.CreateProduct)
	adminGroup.PUT("/products/:id", adminHandler.UpdateProduct)
	adminGroup.DELETE("/products/:id", adminHandler.DeleteProduct)
	adminGroup.GET("/projects", adminHandler.ListProjects)
	adminGroup.POST("/projects", adminHandler.CreateProject)
	adminGroup.PUT("/projects/:id", adminHandler.UpdateProject)
	adminGroup.DELETE("/projects/:id", adminHandler.DeleteProject)
	adminGroup.POST("/projects/:id/bundle", adminHandler.UploadProjectBundle)
	adminGroup.POST("/projects/:id/runtime-bundle", adminHandler.UploadRuntimeBundle)
	adminGroup.GET("/orders", adminHandler.ListOrders)
	adminGroup.PATCH("/orders/:id/status", adminHandler.UpdateOrderStatus)
	adminGroup.GET("/analytics", analyticsHandler.Summary)
	return r
}
