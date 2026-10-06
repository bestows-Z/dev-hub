package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

type ArticleSearch interface {
	Ready() bool
	Disable()
	Search(context.Context, ArticleFilter, int, int) ([]uint64, int64, error)
}

type repository struct {
	db     *gorm.DB
	search ArticleSearch
}

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func NewSearchRepository(db *gorm.DB, search ArticleSearch) Repository {
	return &repository{db: db, search: search}
}

func (r *repository) ListArticles(ctx context.Context, filter ArticleFilter, limit, offset int) ([]Article, int64, error) {
	if strings.TrimSpace(filter.Search) != "" && r.search != nil && r.search.Ready() {
		ids, total, err := r.search.Search(ctx, filter, limit, offset)
		if err == nil {
			if len(ids) == 0 {
				return []Article{}, total, nil
			}
			var articles []Article
			err = r.db.WithContext(ctx).Where("id IN ? AND status = ?", ids, "published").Find(&articles).Error
			if err == nil && len(articles) == len(ids) {
				position := make(map[uint64]int, len(ids))
				for index, id := range ids {
					position[id] = index
				}
				sort.Slice(articles, func(i, j int) bool { return position[articles[i].ID] < position[articles[j].ID] })
				return articles, total, nil
			}
		}
		r.search.Disable()
	}
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
