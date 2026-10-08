package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	logger  *zap.Logger
	codes   EmailVerifier
}

type EmailVerifier interface {
	Verify(context.Context, string, string, string) (bool, error)
}

func (h *Handler) SetEmailVerifier(codes EmailVerifier) { h.codes = codes }

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		RegisterRequest
		EmailCode string `json:"email_code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams,
			"invalid request parameters")
		return
	}
	if h.codes == nil {
		c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50310, Message: "email verification is unavailable"})
		return
	}
	if err := h.service.ValidateRegistration(c.Request.Context(), req.RegisterRequest); err != nil {
		h.handleRegisterError(c, err)
		return
	}
	valid, err := h.codes.Verify(c.Request.Context(), "register", req.Email, req.EmailCode)
	if err != nil {
		h.logger.Error("verify registration email", zap.Error(err))
		response.Error(c)
		return
	}
	if !valid {
		response.Fail(c, 40103, "invalid or expired email code")
		return
	}
	result, err := h.service.Register(c.Request.Context(), req.RegisterRequest)
	if err != nil {
		h.handleRegisterError(c, err)
		return
	}
	response.SuccessMessage(
		c,
		"registered successfully",
		result,
	)
}

func (h *Handler) handleRegisterError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(err, ErrUsernameExists):
		response.Fail(
			c,
			response.CodeUsernameExists,
			"username already exists",
		)

	case errors.Is(err, ErrEmailExists):
		response.Fail(
			c,
			response.CodeEmailExists,
			"email already exists",
		)

	case errors.Is(err, ErrInvalidUsername):
		response.Fail(
			c,
			response.CodeInvalidUsername,
			"username may only contain letters, numbers and underscores",
		)

	case errors.Is(err, ErrPasswordTooLong):
		response.Fail(
			c,
			response.CodePasswordTooLong,
			"password is too long",
		)

	default:
		h.logger.Error(
			"register user failed",
			zap.Error(err),
		)

		response.Error(c)
	}
}
