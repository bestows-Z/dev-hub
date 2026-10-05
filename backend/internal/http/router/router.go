package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/admin"
	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/store"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
)

func New(sqlDB *sql.DB, userHandler *user.Handler, authHandler *auth.Handler, contentHandler *content.Handler, storeHandler *store.Handler, projectHandler *project.Handler, adminHandler *admin.Handler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
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
	v1.GET("/articles", contentHandler.ListArticles)
	v1.GET("/articles/:slug", contentHandler.GetArticle)
	v1.GET("/links", contentHandler.ListLinks)
	v1.GET("/products", storeHandler.ListProducts)
	v1.GET("/products/:slug", storeHandler.GetProduct)
	v1.POST("/orders", storeHandler.CreateOrder)
	v1.GET("/projects", projectHandler.List)
	adminGroup := v1.Group("/admin", authHandler.RequireUser(), authHandler.RequireAdmin())
	adminGroup.GET("/articles", adminHandler.ListArticles)
	adminGroup.POST("/articles", adminHandler.CreateArticle)
	adminGroup.PUT("/articles/:id", adminHandler.UpdateArticle)
	adminGroup.DELETE("/articles/:id", adminHandler.DeleteArticle)
	adminGroup.GET("/links", adminHandler.ListLinks)
	adminGroup.POST("/links", adminHandler.CreateLink)
	adminGroup.PUT("/links/:id", adminHandler.UpdateLink)
	adminGroup.DELETE("/links/:id", adminHandler.DeleteLink)
	adminGroup.GET("/products", adminHandler.ListProducts)
	adminGroup.POST("/products", adminHandler.CreateProduct)
	adminGroup.PUT("/products/:id", adminHandler.UpdateProduct)
	adminGroup.DELETE("/products/:id", adminHandler.DeleteProduct)
	adminGroup.GET("/projects", adminHandler.ListProjects)
	adminGroup.POST("/projects", adminHandler.CreateProject)
	adminGroup.PUT("/projects/:id", adminHandler.UpdateProject)
	adminGroup.DELETE("/projects/:id", adminHandler.DeleteProject)
	adminGroup.GET("/orders", adminHandler.ListOrders)
	adminGroup.PATCH("/orders/:id/status", adminHandler.UpdateOrderStatus)
	return r
}
