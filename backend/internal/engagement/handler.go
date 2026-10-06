package engagement

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/geoip"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errCommentTooSoon = errors.New("comment too soon")
var errApplicationPending = errors.New("application pending")

type Handler struct {
	db      *gorm.DB
	logger  *zap.Logger
	regions *geoip.Resolver
}

func NewHandler(db *gorm.DB, logger *zap.Logger) *Handler { return &Handler{db: db, logger: logger} }

func (h *Handler) SetRegionResolver(regions *geoip.Resolver) { h.regions = regions }

func lockUser(tx *gorm.DB, id uint64) error {
	var locked user.User
	return tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&locked).Error
}

func (h *Handler) article(c *gin.Context) (content.Article, bool) {
	var article content.Article
	if err := h.db.WithContext(c.Request.Context()).Select("id").Where("slug = ? AND status = ?", c.Param("slug"), "published").Take(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Response{Code: 40401, Message: "article not found"})
		} else {
			h.logger.Error("find comment article", zap.Error(err))
			response.Error(c)
		}
		return article, false
	}
	return article, true
}

func (h *Handler) ListComments(c *gin.Context) {
	article, ok := h.article(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	query := h.db.WithContext(c.Request.Context()).Model(&Comment{}).Where("article_comments.article_id = ? AND article_comments.status = ?", article.ID, "approved")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("count comments", zap.Error(err))
		response.Error(c)
		return
	}
	items := make([]CommentView, 0)
	if err := query.Select("article_comments.id, article_comments.body, article_comments.ip_region, users.username, article_comments.reply_to_id, reply_users.username AS reply_to_username, article_comments.created_at").
		Joins("JOIN users ON users.id = article_comments.user_id").
		Joins("LEFT JOIN article_comments reply_target ON reply_target.id = article_comments.reply_to_id").
		Joins("LEFT JOIN users reply_users ON reply_users.id = reply_target.user_id").
		Order("article_comments.created_at DESC, article_comments.id DESC").Limit(20).Offset((page - 1) * 20).Scan(&items).Error; err != nil {
		h.logger.Error("list comments", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": 20})
}

func (h *Handler) CreateComment(c *gin.Context) {
	article, ok := h.article(c)
	if !ok {
		return
	}
	var input struct {
		Body      string  `json:"body" binding:"required"`
		ReplyToID *uint64 `json:"reply_to_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "comment is required")
		return
	}
	input.Body = strings.TrimSpace(input.Body)
	if n := len([]rune(input.Body)); n < 3 || n > 2000 {
		response.Fail(c, response.CodeInvalidParams, "comment must contain 3 to 2000 characters")
		return
	}
	if input.ReplyToID != nil && *input.ReplyToID == 0 {
		response.Fail(c, response.CodeInvalidParams, "invalid reply target")
		return
	}
	u := auth.CurrentUser(c)
	item := Comment{ArticleID: article.ID, UserID: u.ID, ReplyToID: input.ReplyToID, Body: input.Body, Status: "pending"}
	if h.regions != nil {
		item.IPRegion = h.regions.Region(c.ClientIP())
	}
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := lockUser(tx, u.ID); err != nil {
			return err
		}
		if input.ReplyToID != nil {
			var target Comment
			if err := tx.Select("id").Where("id = ? AND article_id = ? AND status = ?", *input.ReplyToID, article.ID, "approved").Take(&target).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errReplyTargetMissing
				}
				return err
			}
		}
		var recent int64
		if err := tx.Model(&Comment{}).Where("user_id = ? AND created_at > ?", u.ID, time.Now().Add(-30*time.Second)).Count(&recent).Error; err != nil {
			return err
		}
		if recent > 0 {
			return errCommentTooSoon
		}
		return tx.Create(&item).Error
	})
	if errors.Is(err, errCommentTooSoon) {
		response.Fail(c, 42901, "please wait before commenting again")
		return
	}
	if errors.Is(err, errReplyTargetMissing) {
		response.Fail(c, 40404, "reply target is unavailable")
		return
	}
	if err != nil {
		h.logger.Error("create comment", zap.Error(err))
		response.Error(c)
		return
	}
	response.SuccessMessage(c, "comment submitted for review", gin.H{"id": item.ID, "status": item.Status})
}

var errReplyTargetMissing = errors.New("reply target missing or not approved")

func validURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func (h *Handler) ApplyLink(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		URL         string `json:"url" binding:"required"`
		AvatarURL   string `json:"avatar_url"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid link application")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	input.AvatarURL = strings.TrimSpace(input.AvatarURL)
	input.Description = strings.TrimSpace(input.Description)
	if len([]rune(input.Name)) > 100 || input.Name == "" || !validURL(input.URL) || (input.AvatarURL != "" && !validURL(input.AvatarURL)) || len([]rune(input.Description)) > 280 {
		response.Fail(c, response.CodeInvalidParams, "invalid link application")
		return
	}
	u := auth.CurrentUser(c)
	item := LinkApplication{UserID: u.ID, Name: input.Name, URL: input.URL, AvatarURL: input.AvatarURL, Description: input.Description, Status: "pending"}
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := lockUser(tx, u.ID); err != nil {
			return err
		}
		var pending int64
		if err := tx.Model(&LinkApplication{}).Where("user_id = ? AND status = ?", u.ID, "pending").Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return errApplicationPending
		}
		return tx.Create(&item).Error
	})
	if errors.Is(err, errApplicationPending) {
		response.Fail(c, 40901, "an application is already under review")
		return
	}
	if err != nil {
		h.logger.Error("create link application", zap.Error(err))
		response.Error(c)
		return
	}
	response.SuccessMessage(c, "link application submitted for review", gin.H{"id": item.ID, "status": item.Status})
}
