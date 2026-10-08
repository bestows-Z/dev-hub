package emailcode

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	users   user.Repository
	logger  *zap.Logger
}

func NewHandler(service *Service, users user.Repository, logger *zap.Logger) *Handler {
	return &Handler{service: service, users: users, logger: logger}
}

func ValidEmail(value string) bool {
	if value == "" || len(value) > 255 || strings.ContainsAny(value, "\r\n ") {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func (h *Handler) Request(c *gin.Context) {
	h.request(c, "")
}

// RequestChange is registered behind the authenticated-user middleware.
func (h *Handler) RequestChange(c *gin.Context) {
	h.request(c, "change_email")
}

func (h *Handler) request(c *gin.Context, forcedPurpose string) {
	var input struct {
		Email   string `json:"email" binding:"required"`
		Purpose string `json:"purpose"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&input); err != nil || !ValidEmail(input.Email) {
		response.Fail(c, response.CodeInvalidParams, "invalid email or purpose")
		return
	}
	if forcedPurpose != "" {
		input.Purpose = forcedPurpose
	} else if input.Purpose != "register" && input.Purpose != "login" {
		response.Fail(c, response.CodeInvalidParams, "invalid email or purpose")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	exists, err := h.users.ExistsByEmail(c.Request.Context(), input.Email)
	if err != nil {
		h.logger.Error("check verification email", zap.Error(err))
		response.Error(c)
		return
	}
	if input.Purpose != "login" && exists {
		response.Fail(c, response.CodeEmailExists, "email already exists")
		return
	}
	if input.Purpose == "login" && !exists {
		response.Success(c, gin.H{"sent": true, "expires_in": 600, "cooldown": 60})
		return
	}
	err = h.service.Request(c.Request.Context(), input.Purpose, input.Email, c.ClientIP())
	if errors.Is(err, ErrTooFrequent) {
		c.JSON(http.StatusTooManyRequests, response.Response{Code: 42902, Message: "wait before requesting another code"})
		return
	}
	if err != nil {
		h.logger.Error("send verification email", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50310, Message: "verification email is temporarily unavailable"})
		return
	}
	response.Success(c, gin.H{"sent": true, "expires_in": 600, "cooldown": 60})
}
