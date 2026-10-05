package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// usernameRegexp:
//
// 允许：
// tianhao
// tian_hao
// tianhao123
//
// 不允许：
// tian-hao
// 张三
// tian hao
var usernameRegexp = regexp.MustCompile(
	`^[a-zA-Z0-9_]+$`,
)

// Service 负责 User 业务逻辑。
type Service struct {
	repository Repository
}

// NewService 创建 User Service。
func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

// Register 注册用户。
func (s *Service) Register(
	ctx context.Context,
	req RegisterRequest,
) (*UserResponse, error) {

	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(
		strings.TrimSpace(req.Email),
	)

	// 用户名统一使用小写
	username = strings.ToLower(username)

	if !usernameRegexp.MatchString(username) {
		return nil, ErrInvalidUsername
	}
	if len([]byte(req.Password)) > 72 {
		return nil, ErrPasswordTooLong
	}

	exists, err := s.repository.ExistsByUsername(
		ctx,
		username,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"check username: %w",
			err,
		)
	}

	if exists {
		return nil, ErrUsernameExists
	}

	exists, err = s.repository.ExistsByEmail(
		ctx,
		email,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"check email: %w",
			err,
		)
	}

	if exists {
		return nil, ErrEmailExists
	}

	passwordHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"hash password: %w",
			err,
		)
	}

	u := &User{
		Username: username,
		Email:    email,

		PasswordHash: string(passwordHash),

		Role:   RoleUser,
		Status: StatusNormal,
	}

	if err := s.repository.Create(
		ctx,
		u,
	); err != nil {

		return nil, err
	}

	response := ToResponse(u)

	return &response, nil
}
