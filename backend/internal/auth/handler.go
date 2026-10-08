package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/geoip"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	users   user.Repository
	tokens  *Tokens
	logger  *zap.Logger
	db      *gorm.DB
	store   *storage.Store
	regions *geoip.Resolver
	codes   user.EmailVerifier
	oauth   *oauthService
}

func (h *Handler) SetRegionResolver(regions *geoip.Resolver) { h.regions = regions }
func (h *Handler) SetEmailVerifier(codes user.EmailVerifier) { h.codes = codes }

func NewHandler(users user.Repository, tokens *Tokens, db *gorm.DB, store *storage.Store, logger *zap.Logger) *Handler {
	return &Handler{users: users, tokens: tokens, db: db, store: store, logger: logger}
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid login details")
		return
	}
	u, err := h.users.FindByIdentifier(c.Request.Context(), strings.TrimSpace(req.Identifier))
	if errors.Is(err, user.ErrNotFound) {
		response.Fail(c, 40101, "invalid credentials")
		return
	}
	if err != nil {
		h.logger.Error("find login user", zap.Error(err))
		response.Error(c)
		return
	}
	if u.Status != user.StatusNormal || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		response.Fail(c, 40101, "invalid credentials")
		return
	}
	h.completeLogin(c, u)
}

func (h *Handler) LoginEmail(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email,max=255"`
		EmailCode string `json:"email_code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid email login details")
		return
	}
	if h.codes == nil {
		c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50310, Message: "email verification is unavailable"})
		return
	}
	valid, err := h.codes.Verify(c.Request.Context(), "login", req.Email, req.EmailCode)
	if err != nil {
		h.logger.Error("verify login email", zap.Error(err))
		response.Error(c)
		return
	}
	if !valid {
		response.Fail(c, 40101, "invalid or expired email code")
		return
	}
	u, err := h.users.FindByIdentifier(c.Request.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil || u.Status != user.StatusNormal {
		response.Fail(c, 40101, "invalid or expired email code")
		return
	}
	h.completeLogin(c, u)
}

func (h *Handler) completeLogin(c *gin.Context, u *user.User) {
	if h.regions != nil {
		region := h.regions.Region(c.ClientIP())
		if err := h.db.WithContext(c.Request.Context()).Model(u).Update("last_ip_region", region).Error; err != nil {
			h.logger.Warn("update login IP region", zap.Error(err))
		} else {
			u.LastIPRegion = region
		}
	}
	token, err := h.tokens.Issue(u.ID)
	if err != nil {
		h.logger.Error("issue token", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": 43200, "user": user.ToResponse(u)})
}

const currentUserKey = "currentUser"

func CurrentUser(c *gin.Context) *user.User {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return nil
	}
	u, _ := value.(*user.User)
	return u
}

func (h *Handler) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, response.Response{Code: 40102, Message: "authentication required"})
			c.Abort()
			return
		}
		id, err := h.tokens.Verify(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.Response{Code: 40102, Message: "invalid access token"})
			c.Abort()
			return
		}
		u, err := h.users.FindByID(c.Request.Context(), id)
		if err != nil || u.Status != user.StatusNormal {
			c.JSON(http.StatusUnauthorized, response.Response{Code: 40102, Message: "invalid access token"})
			c.Abort()
			return
		}
		c.Set(currentUserKey, u)
		c.Next()
	}
}

func (h *Handler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(currentUserKey)
		if !exists || value.(*user.User).Role != user.RoleAdmin {
			c.JSON(http.StatusForbidden, response.Response{Code: 40301, Message: "administrator access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) RequireAuthor() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || (u.Role != user.RoleAdmin && u.Role != user.RoleAuthor) {
			c.JSON(http.StatusForbidden, response.Response{Code: 40302, Message: "author access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) Me(c *gin.Context) {
	value, _ := c.Get(currentUserKey)
	response.Success(c, user.ToResponse(value.(*user.User)))
}
