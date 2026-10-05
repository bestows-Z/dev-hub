package user

import (
	"errors"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams,
			"invalid request parameters")
		return
	}
	result, err := h.service.Register(c.Request.Context(), req)
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
