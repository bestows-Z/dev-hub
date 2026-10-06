package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"gorm.io/gorm"
)

const articleIndex = "devhub_articles_v1"

type Client struct {
	baseURL string
	index   string
	http    *http.Client
	ready   atomic.Bool
}

func New(rawURL string) *Client {
	return &Client{baseURL: strings.TrimRight(rawURL, "/"), index: articleIndex, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Ready() bool { return c.ready.Load() }
func (c *Client) Disable()    { c.ready.Store(false) }

func (c *Client) request(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return nil, resp.StatusCode, fmt.Errorf("elasticsearch %s %s: HTTP %d: %s", method, path, resp.StatusCode, string(data))
	}
	return data, resp.StatusCode, nil
}

func (c *Client) Rebuild(ctx context.Context, db *gorm.DB) error {
	c.Disable()
	path := "/" + c.index
	if _, _, err := c.request(ctx, http.MethodDelete, path, nil); err != nil {
		return err
	}
	if err := c.createIndex(ctx); err != nil {
		return err
	}
	var lastID uint64
	for {
		var articles []content.Article
		if err := db.WithContext(ctx).Where("status = ? AND id > ?", "published", lastID).Order("id ASC").Limit(100).Find(&articles).Error; err != nil {
			return err
		}
		for _, article := range articles {
			if err := c.upsert(ctx, article, false); err != nil {
				return err
			}
			lastID = article.ID
		}
		if len(articles) < 100 {
			break
		}
	}
	if _, _, err := c.request(ctx, http.MethodPost, path+"/_refresh", nil); err != nil {
		return err
	}
	c.ready.Store(true)
	return nil
}

func (c *Client) createIndex(ctx context.Context) error {
	mapping := map[string]any{"mappings": map[string]any{"properties": map[string]any{
		"title":        map[string]any{"type": "text", "analyzer": "cjk"},
		"excerpt":      map[string]any{"type": "text", "analyzer": "cjk"},
		"body_md":      map[string]any{"type": "text", "analyzer": "cjk"},
		"category":     map[string]any{"type": "keyword"},
		"tags":         map[string]any{"type": "keyword"},
		"published_at": map[string]any{"type": "date"},
	}}}
	_, _, err := c.request(ctx, http.MethodPut, "/"+c.index, mapping)
	return err
}

func (c *Client) upsert(ctx context.Context, article content.Article, refresh bool) error {
	path := "/" + c.index + "/_doc/" + strconv.FormatUint(article.ID, 10)
	if refresh {
		path += "?refresh=wait_for"
	}
	doc := map[string]any{"title": article.Title, "excerpt": article.Excerpt, "body_md": article.BodyMD, "category": article.Category, "tags": article.Tags, "published_at": article.PublishedAt}
	_, _, err := c.request(ctx, http.MethodPut, path, doc)
	return err
}

func (c *Client) SyncArticle(ctx context.Context, article content.Article) error {
	if article.Status != "published" {
		return c.DeleteArticle(ctx, article.ID)
	}
	return c.upsert(ctx, article, true)
}

func (c *Client) DeleteArticle(ctx context.Context, id uint64) error {
	_, _, err := c.request(ctx, http.MethodDelete, "/"+c.index+"/_doc/"+strconv.FormatUint(id, 10)+"?refresh=wait_for", nil)
	return err
}

func (c *Client) Search(ctx context.Context, filter content.ArticleFilter, limit, offset int) ([]uint64, int64, error) {
	filters := []any{}
	if filter.Category != "" {
		filters = append(filters, map[string]any{"term": map[string]any{"category": filter.Category}})
	}
	if filter.Tag != "" {
		filters = append(filters, map[string]any{"term": map[string]any{"tags": filter.Tag}})
	}
	query := map[string]any{"bool": map[string]any{"must": []any{map[string]any{"multi_match": map[string]any{"query": strings.TrimSpace(filter.Search), "fields": []string{"title^4", "excerpt^2", "body_md"}}}}, "filter": filters}}
	request := map[string]any{"query": query, "from": offset, "size": limit, "track_total_hits": true, "_source": false, "sort": []any{"_score", map[string]any{"published_at": "desc"}}}
	data, _, err := c.request(ctx, http.MethodPost, "/"+c.index+"/_search", request)
	if err != nil {
		return nil, 0, err
	}
	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		id, err := strconv.ParseUint(hit.ID, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	return ids, result.Hits.Total.Value, nil
}

func (c *Client) URL() string {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
