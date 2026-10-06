package user

import "time"

type Role int16

const (
	RoleAdmin Role = iota + 1
	RoleUser
	RoleAuthor
)

type Status int16

const (
	StatusNormal Status = iota + 1
	StatusDisabled
)

type User struct {
	ID uint64 `gorm:"column:id;primaryKey"`

	Username       string `gorm:"column:username"`
	Email          string `gorm:"column:email"`
	PasswordHash   string `gorm:"column:password_hash"`
	DisplayName    string `gorm:"column:display_name"`
	Bio            string `gorm:"column:bio"`
	WebsiteURL     string `gorm:"column:website_url"`
	AvatarUploaded bool   `gorm:"column:avatar_uploaded"`

	Role   Role   `gorm:"column:role"`
	Status Status `gorm:"column:status"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}
