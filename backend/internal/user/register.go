package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// RegisterRequest is the public registration payload.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8"`
}

// Register keeps both user identifiers in canonical lowercase form.
func (s *Service) ValidateRegistration(ctx context.Context, req RegisterRequest) error {
	username := strings.ToLower(strings.TrimSpace(req.Username))
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if len(username) < 3 || len(username) > 32 || !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	if len([]byte(req.Password)) > 72 {
		return ErrPasswordTooLong
	}
	if len(req.Password) < 8 {
		return fmt.Errorf("password must have at least 8 bytes")
	}

	exists, err := s.repository.ExistsByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameExists
	}
	exists, err = s.repository.ExistsByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if exists {
		return ErrEmailExists
	}
	return nil
}

// Register rechecks identifiers; database uniqueness handles concurrent requests.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*Response, error) {
	if err := s.ValidateRegistration(ctx, req); err != nil {
		return nil, err
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	email := strings.ToLower(strings.TrimSpace(req.Email))

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         RoleUser,
		Status:       StatusNormal,
	}
	if err := s.repository.Create(ctx, u); err != nil {
		return nil, err
	}
	result := ToResponse(u)
	return &result, nil
}
