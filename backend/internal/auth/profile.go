package auth

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const maxAvatarBytes = 2 << 20

type profileInput struct {
	Email       string `json:"email" binding:"required,email,max=255"`
	EmailCode   string `json:"email_code" binding:"omitempty,len=6"`
	DisplayName string `json:"display_name" binding:"max=60"`
	Bio         string `json:"bio" binding:"max=500"`
	WebsiteURL  string `json:"website_url" binding:"max=255"`
}

func validWebsite(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

func validProfileLengths(input profileInput) bool {
	return utf8.RuneCountInString(input.Email) <= 255 &&
		utf8.RuneCountInString(input.DisplayName) <= 60 &&
		utf8.RuneCountInString(input.Bio) <= 500 &&
		utf8.RuneCountInString(input.WebsiteURL) <= 255
}

func (h *Handler) UpdateMe(c *gin.Context) {
	var input profileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, response.CodeInvalidParams, "invalid profile details")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Bio = strings.TrimSpace(input.Bio)
	input.WebsiteURL = strings.TrimSpace(input.WebsiteURL)
	if !validProfileLengths(input) || !validWebsite(input.WebsiteURL) {
		response.Fail(c, response.CodeInvalidParams, "invalid profile details")
		return
	}
	current := CurrentUser(c)
	if input.Email != current.Email {
		if h.codes == nil {
			c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50310, Message: "email verification is unavailable"})
			return
		}
		valid, err := h.codes.Verify(c.Request.Context(), "change_email", input.Email, input.EmailCode)
		if err != nil {
			h.logger.Error("verify profile email", zap.Error(err))
			response.Error(c)
			return
		}
		if !valid {
			response.Fail(c, 40103, "invalid or expired email code")
			return
		}
	}
	id := current.ID
	err := h.db.WithContext(c.Request.Context()).Model(&user.User{}).Where("id = ?", id).Updates(map[string]any{
		"email": input.Email, "display_name": input.DisplayName, "bio": input.Bio, "website_url": input.WebsiteURL,
	}).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_users_email" {
			response.Fail(c, response.CodeEmailExists, "email already exists")
			return
		}
		h.logger.Error("update profile", zap.Error(err))
		response.Error(c)
		return
	}
	h.respondFreshUser(c, id)
}

func decodeAvatar(blob []byte) ([]byte, error) {
	if len(blob) == 0 || len(blob) > maxAvatarBytes {
		return nil, errors.New("avatar must be 1 byte to 2 MiB")
	}
	mime := http.DetectContentType(blob)
	if mime != "image/png" && mime != "image/jpeg" {
		return nil, errors.New("avatar must be PNG or JPEG")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(blob))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 2048 || config.Height > 2048 || int64(config.Width)*int64(config.Height) > 4_000_000 {
		return nil, errors.New("avatar dimensions must be within 2048 pixels and 4 million pixels total")
	}
	img, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return nil, errors.New("avatar image cannot be decoded")
	}
	var normalized bytes.Buffer
	if err := png.Encode(&normalized, img); err != nil {
		return nil, err
	}
	return normalized.Bytes(), nil
}

func avatarKey(id uint64) string { return fmt.Sprintf("avatars/%d.png", id) }

func (h *Handler) UploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarBytes+(1<<20))
	file, _, err := c.Request.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, "choose a PNG or JPEG up to 2 MiB")
		return
	}
	defer file.Close()
	blob, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, "cannot read avatar")
		return
	}
	imageBody, err := decodeAvatar(blob)
	if err != nil {
		response.Fail(c, response.CodeInvalidParams, err.Error())
		return
	}
	id := CurrentUser(c).ID
	if err := h.store.Put(c.Request.Context(), avatarKey(id), imageBody, "image/png"); err != nil {
		h.logger.Error("store avatar", zap.Error(err))
		response.Error(c)
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Model(&user.User{}).Where("id = ?", id).Updates(map[string]any{"avatar_uploaded": true}).Error; err != nil {
		h.logger.Error("update avatar profile", zap.Error(err))
		response.Error(c)
		return
	}
	h.respondFreshUser(c, id)
}

func (h *Handler) DeleteAvatar(c *gin.Context) {
	id := CurrentUser(c).ID
	if err := h.db.WithContext(c.Request.Context()).Model(&user.User{}).Where("id = ?", id).Updates(map[string]any{"avatar_uploaded": false}).Error; err != nil {
		h.logger.Error("clear avatar profile", zap.Error(err))
		response.Error(c)
		return
	}
	if err := h.store.Remove(c.Request.Context(), avatarKey(id)); err != nil && !storage.IsNotFound(err) {
		h.logger.Error("remove avatar", zap.Error(err))
		response.Error(c)
		return
	}
	h.respondFreshUser(c, id)
}

func (h *Handler) respondFreshUser(c *gin.Context, id uint64) {
	u, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("read updated profile", zap.Error(err))
		response.Error(c)
		return
	}
	response.Success(c, user.ToResponse(u))
}

func (h *Handler) ServeAvatar(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	u, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil || !u.AvatarUploaded {
		c.Status(http.StatusNotFound)
		return
	}
	body, size, _, err := h.store.Get(c.Request.Context(), avatarKey(id))
	if err != nil {
		if !storage.IsNotFound(err) {
			h.logger.Error("read avatar", zap.Error(err))
		}
		c.Status(http.StatusNotFound)
		return
	}
	defer body.Close()
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "public, max-age=300")
	c.DataFromReader(http.StatusOK, size, "image/png", body, nil)
}
