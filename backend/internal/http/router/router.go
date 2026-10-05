package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/store"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
)

func New(sqlDB *sql.DB, userHandler *user.Handler, contentHandler *content.Handler, storeHandler *store.Handler, projectHandler *project.Handler) *gin.Engine {
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
	}
	v1.GET("/articles", contentHandler.ListArticles)
	v1.GET("/articles/:slug", contentHandler.GetArticle)
	v1.GET("/links", contentHandler.ListLinks)
	v1.GET("/products", storeHandler.ListProducts)
	v1.GET("/products/:slug", storeHandler.GetProduct)
	v1.POST("/orders", storeHandler.CreateOrder)
	v1.GET("/projects", projectHandler.List)
	return r
}
