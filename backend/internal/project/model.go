package project

import "time"

type Project struct {
	ID                  uint64    `json:"id" gorm:"primaryKey"`
	Slug                string    `json:"slug"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	CoverURL            string    `json:"cover_url"`
	Tags                []string  `json:"tags" gorm:"type:jsonb;serializer:json"`
	PreviewURL          string    `json:"preview_url"`
	BackendURL          string    `json:"backend_url"`
	BundlePrefix        string    `json:"-"`
	RuntimeStatus       string    `json:"runtime_status"`
	RuntimeFrontendPort int       `json:"-"`
	RuntimeBackendPort  int       `json:"-"`
	SourceURL           string    `json:"source_url"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}
