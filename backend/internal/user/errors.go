package user

import "errors"

var (
	// ErrUsernameExists 用户名已经存在。
	ErrUsernameExists = errors.New("username already exists")

	// ErrEmailExists 邮箱已经存在。
	ErrEmailExists = errors.New("email already exists")

	// ErrInvalidUsername 用户名格式不合法。
	ErrInvalidUsername = errors.New("invalid username")

	// ErrPasswordTooLong bcrypt 最大只接受 72 字节密码。
	ErrPasswordTooLong = errors.New("password is too long")
)
