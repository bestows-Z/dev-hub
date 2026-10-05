package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrArticleNotFound = errors.New("article not found")

type Repository interface {
	ListArticles(context.Context, ArticleFilter, int, int) ([]Article, int64, error)
	ArticleFacets(context.Context) (ArticleFacets, error)
	GetArticle(context.Context, string) (*Article, error)
	ListLinks(context.Context) ([]FriendLink, error)
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) ListArticles(ctx context.Context, filter ArticleFilter, limit, offset int) ([]Article, int64, error) {
	query := r.db.WithContext(ctx).Model(&Article{}).Where("status = ?", "published")
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where("title ILIKE ? OR excerpt ILIKE ? OR body_md ILIKE ?", like, like, like)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Tag != "" {
		tag, _ := json.Marshal([]string{filter.Tag})
		query = query.Where("tags @> ?::jsonb", string(tag))
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

func (r *repository) ArticleFacets(ctx context.Context) (ArticleFacets, error) {
	result := ArticleFacets{Categories: []ArticleFacet{}, Tags: []ArticleFacet{}}
	if err := r.db.WithContext(ctx).Raw("SELECT category AS name, COUNT(*) AS count FROM articles WHERE status = 'published' GROUP BY category ORDER BY count DESC, name").Scan(&result.Categories).Error; err != nil {
		return ArticleFacets{}, fmt.Errorf("article category facets: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw("SELECT tag AS name, COUNT(*) AS count FROM articles, LATERAL jsonb_array_elements_text(tags) AS tag WHERE status = 'published' GROUP BY tag ORDER BY count DESC, name LIMIT 40").Scan(&result.Tags).Error; err != nil {
		return ArticleFacets{}, fmt.Errorf("article tag facets: %w", err)
	}
	return result, nil
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
