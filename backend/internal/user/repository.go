package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Repository 定义 User 数据访问行为。
//
// Service 不应该关心底层到底是：
// PostgreSQL、MySQL 还是其他数据库。
type Repository interface {
	ExistsByUsername(
		ctx context.Context,
		username string,
	) (bool, error)

	ExistsByEmail(
		ctx context.Context,
		email string,
	) (bool, error)

	Create(
		ctx context.Context,
		user *User,
	) error
}

// repository 是 PostgreSQL + GORM 的具体实现。
type repository struct {
	db *gorm.DB
}

// NewRepository 创建 User Repository。
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// ExistsByUsername 判断用户名是否存在。
func (r *repository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	var u User

	err := r.db.
		WithContext(ctx).
		Select("id").
		Where("username = ?", username).
		Take(&u).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf(
			"find user by username: %w",
			err,
		)
	}

	return true, nil
}

// ExistsByEmail 判断邮箱是否存在。
func (r *repository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	var u User

	err := r.db.
		WithContext(ctx).
		Select("id").
		Where("email = ?", email).
		Take(&u).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf(
			"find user by email: %w",
			err,
		)
	}

	return true, nil
}

// Create 创建用户。
func (r *repository) Create(
	ctx context.Context,
	u *User,
) error {
	err := r.db.
		WithContext(ctx).
		Create(u).
		Error

	if err == nil {
		return nil
	}

	// PostgreSQL SQLSTATE 23505：
	// unique_violation。
	//
	// 为什么 Service 已经检查用户名/邮箱了，
	// 这里仍然必须处理？
	//
	// 因为并发情况下两个请求可能同时通过 Exists 检查。
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" {

		switch pgErr.ConstraintName {

		case "uq_users_username":
			return ErrUsernameExists

		case "uq_users_email":
			return ErrEmailExists
		}
	}

	return fmt.Errorf(
		"create user: %w",
		err,
	)
}
