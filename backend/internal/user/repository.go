package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

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

	FindByIdentifier(
		ctx context.Context,
		identifier string,
	) (*User, error)

	FindByID(
		ctx context.Context,
		id uint64,
	) (*User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	var count int64

	err := r.db.
		WithContext(ctx).
		Model(&User{}).
		Where("username = ?", username).
		Count(&count).
		Error

	if err != nil {
		return false, fmt.Errorf(
			"check username exists: %w",
			err,
		)
	}

	return count > 0, nil
}

func (r *repository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	var count int64

	err := r.db.
		WithContext(ctx).
		Model(&User{}).
		Where("email = ?", email).
		Count(&count).
		Error

	if err != nil {
		return false, fmt.Errorf(
			"check email exists: %w",
			err,
		)
	}

	return count > 0, nil
}

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

// FindByIdentifier 支持使用用户名或邮箱登录。
func (r *repository) FindByIdentifier(
	ctx context.Context,
	identifier string,
) (*User, error) {
	var u User

	identifier = strings.ToLower(
		strings.TrimSpace(identifier),
	)

	err := r.db.
		WithContext(ctx).
		Where(
			"username = ? OR email = ?",
			identifier,
			identifier,
		).
		Take(&u).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find user by identifier: %w",
			err,
		)
	}

	return &u, nil
}

func (r *repository) FindByID(
	ctx context.Context,
	id uint64,
) (*User, error) {
	var u User

	err := r.db.
		WithContext(ctx).
		First(&u, id).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"find user by id: %w",
			err,
		)
	}

	return &u, nil
}
