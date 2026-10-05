package content

import "time"

type Article struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	BodyMD      string     `json:"body_md" gorm:"column:body_md"`
	CoverURL    string     `json:"cover_url"`
	Tags        []string   `json:"tags" gorm:"type:jsonb;serializer:json"`
	Status      string     `json:"-"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
}

type ArticleSummary struct {
	ID          uint64     `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	CoverURL    string     `json:"cover_url"`
	Tags        []string   `json:"tags"`
	PublishedAt *time.Time `json:"published_at"`
}

func (a Article) Summary() ArticleSummary {
	return ArticleSummary{a.ID, a.Slug, a.Title, a.Excerpt, a.CoverURL, a.Tags, a.PublishedAt}
}

type FriendLink struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	AvatarURL   string    `json:"avatar_url"`
	Description string    `json:"description"`
	SortOrder   int       `json:"-"`
	Enabled     bool      `json:"-"`
	CreatedAt   time.Time `json:"-"`
}

func (FriendLink) TableName() string { return "friend_links" }
