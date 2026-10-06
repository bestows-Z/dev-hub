package user

import (
	"fmt"
	"time"
)

type Response struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	WebsiteURL  string `json:"website_url"`
	AvatarURL   string `json:"avatar_url"`

	Role   Role   `json:"role"`
	Status Status `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

func ToResponse(u *User) Response {
	avatarURL := ""
	if u.AvatarUploaded {
		avatarURL = fmt.Sprintf("/api/v1/avatars/%d?v=%d", u.ID, u.UpdatedAt.UnixNano())
	}
	return Response{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		WebsiteURL:  u.WebsiteURL,
		AvatarURL:   avatarURL,
		Role:        u.Role,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
	}
}
