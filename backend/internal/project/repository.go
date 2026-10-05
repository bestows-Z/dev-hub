package project

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Repository interface {
	List(context.Context, int, int) ([]Project, int64, error)
}
type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db} }

func (r *repository) List(ctx context.Context, limit, offset int) ([]Project, int64, error) {
	q := r.db.WithContext(ctx).Model(&Project{}).Where("status = ?", "published")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}
	projects := make([]Project, 0)
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	return projects, total, nil
}
