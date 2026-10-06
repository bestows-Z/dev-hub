package engagement

import "time"

type Comment struct {
	ID         uint64     `json:"id" gorm:"primaryKey"`
	ArticleID  uint64     `json:"article_id"`
	UserID     uint64     `json:"user_id"`
	ReplyToID  *uint64    `json:"reply_to_id"`
	Body       string     `json:"body"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ReviewedAt *time.Time `json:"reviewed_at"`
}

func (Comment) TableName() string { return "article_comments" }

type CommentView struct {
	ID              uint64    `json:"id"`
	Body            string    `json:"body"`
	Username        string    `json:"username"`
	ReplyToID       *uint64   `json:"reply_to_id"`
	ReplyToUsername string    `json:"reply_to_username"`
	CreatedAt       time.Time `json:"created_at"`
}

type LinkApplication struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	UserID      uint64     `json:"user_id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	AvatarURL   string     `json:"avatar_url"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
}

func (LinkApplication) TableName() string { return "link_applications" }
