package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthorApplication struct {
	ID         uint64     `json:"id" gorm:"primaryKey"`
	UserID     uint64     `json:"user_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	Username   string     `json:"username" gorm:"->;-:migration"`
	Email      string     `json:"email" gorm:"->;-:migration"`
}

func (h *Handler) ApplyAuthor(c *gin.Context) {
	u := auth.CurrentUser(c)
	if u.Role != user.RoleUser {
		response.Fail(c, 40920, "this account already has writing access")
		return
	}
	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len([]rune(clean(input.Reason))) < 10 || len([]rune(clean(input.Reason))) > 1000 {
		response.Fail(c, response.CodeInvalidParams, "tell us why you want to write here (10–1000 characters)")
		return
	}
	item := AuthorApplication{UserID: u.ID, Reason: clean(input.Reason), Status: "pending"}
	if err := h.db.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Fail(c, 40921, "your application is already pending")
			return
		}
		h.failure(c, "apply for author", err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) MyAuthorApplications(c *gin.Context) {
	u := auth.CurrentUser(c)
	items := make([]AuthorApplication, 0)
	if err := h.db.WithContext(c.Request.Context()).Where("user_id = ?", u.ID).Order("created_at DESC, id DESC").Limit(10).Find(&items).Error; err != nil {
		h.failure(c, "list my author applications", err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) ListAuthorApplications(c *gin.Context) {
	p, size := page(c)
	q := h.db.WithContext(c.Request.Context()).Model(&AuthorApplication{})
	if status := c.Query("status"); status != "" {
		if status != "pending" && status != "approved" && status != "rejected" {
			response.Fail(c, response.CodeInvalidParams, "invalid status")
			return
		}
		q = q.Where("status = ?", status)
	}
	if pattern := searchPattern(c); pattern != "" {
		q = q.Where("user_id IN (SELECT id FROM users WHERE username ILIKE ? OR email ILIKE ?)", pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		h.failure(c, "count author applications", err)
		return
	}
	items := make([]AuthorApplication, 0)
	if err := q.Select("author_applications.*, users.username, users.email").Joins("JOIN users ON users.id = author_applications.user_id").Order("author_applications.created_at ASC, author_applications.id ASC").Limit(size).Offset((p - 1) * size).Find(&items).Error; err != nil {
		h.failure(c, "list author applications", err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": p, "page_size": size})
}

var errApplicationReviewed = errors.New("application already reviewed")
var errApplicantIneligible = errors.New("applicant is no longer eligible")

func (h *Handler) ReviewAuthorApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status" binding:"required,oneof=approved rejected"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid review decision")
		return
	}
	var item AuthorApplication
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if item.Status != "pending" {
			return errApplicationReviewed
		}
		var applicant user.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&applicant, item.UserID).Error; err != nil {
			return err
		}
		if applicant.Role != user.RoleUser || applicant.Status != user.StatusNormal {
			return errApplicantIneligible
		}
		now := time.Now()
		if input.Status == "approved" {
			if err := tx.Model(&applicant).Update("role", user.RoleAuthor).Error; err != nil {
				return err
			}
		}
		return tx.Model(&item).Updates(map[string]any{"status": input.Status, "reviewed_at": now}).Error
	})
	if errors.Is(err, errApplicationReviewed) || errors.Is(err, errApplicantIneligible) {
		c.JSON(http.StatusConflict, response.Response{Code: 40922, Message: "application can no longer be reviewed"})
		return
	}
	if err != nil {
		h.failure(c, "review author application", err)
		return
	}
	response.Success(c, item)
}
