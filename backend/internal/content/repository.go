package content

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrArticleNotFound = errors.New("article not found")

type Repository interface {
	ListArticles(context.Context, string, int, int) ([]Article, int64, error)
	GetArticle(context.Context, string) (*Article, error)
	ListLinks(context.Context) ([]FriendLink, error)
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) ListArticles(ctx context.Context, search string, limit, offset int) ([]Article, int64, error) {
	query := r.db.WithContext(ctx).Model(&Article{}).Where("status = ?", "published")
	if search = strings.TrimSpace(search); search != "" {
		query = query.Where("title ILIKE ? OR excerpt ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}
	articles := make([]Article, 0)
	if err := query.Order("published_at DESC, id DESC").Limit(limit).Offset(offset).Find(&articles).Error; err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}
	return articles, total, nil
}

func (r *repository) GetArticle(ctx context.Context, slug string) (*Article, error) {
	var article Article
	err := r.db.WithContext(ctx).Where("slug = ? AND status = ?", slug, "published").Take(&article).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}
	return &article, nil
}

func (r *repository) ListLinks(ctx context.Context) ([]FriendLink, error) {
	links := make([]FriendLink, 0)
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("sort_order ASC, id ASC").Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	return links, nil
}
