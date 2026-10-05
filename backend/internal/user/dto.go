package user

import "time"

type Response struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	Role   Role   `json:"role"`
	Status Status `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

func ToResponse(u *User) Response {
	return Response{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}
