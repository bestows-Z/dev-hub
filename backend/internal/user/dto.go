package user

import "time"

// RegisterRequest 注册请求。
//
// binding 由 Gin 内置的 validator 进行基础参数校验。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// UserResponse 对外返回的用户数据。
//
// 注意这里绝对没有 PasswordHash。
type UserResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	Role   Role   `json:"role"`
	Status Status `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

// ToResponse 将数据库 Model 转成 API DTO。
func ToResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}
