package search

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/content"
)

func TestArticleSearchIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_ES") != "1" {
		t.Skip("set DEVHUB_TEST_ES=1 to test against local Elasticsearch")
	}
	endpoint := os.Getenv("ELASTICSEARCH_URL")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9200"
	}
	client := New(endpoint)
	client.index = "devhub_article_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ctx := context.Background()
	defer client.request(ctx, http.MethodDelete, "/"+client.index, nil)
	if err := client.createIndex(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	article := content.Article{ID: 1, Title: "兔子写数据库笔记", Excerpt: "PostgreSQL 事务", BodyMD: "数据库的事务可以保护订单库存。", Category: "tech", Tags: []string{"数据库"}, Status: "published", PublishedAt: &now}
	if err := client.SyncArticle(ctx, article); err != nil {
		t.Fatal(err)
	}
	ids, total, err := client.Search(ctx, content.ArticleFilter{Search: "数据库", Category: "tech", Tag: "数据库"}, 10, 0)
	if err != nil || total != 1 || len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("search result: ids=%v total=%d error=%v", ids, total, err)
	}
	ids, total, err = client.Search(ctx, content.ArticleFilter{Search: "数据库", Category: "travel"}, 10, 0)
	if err != nil || total != 0 || len(ids) != 0 {
		t.Fatalf("category filter: ids=%v total=%d error=%v", ids, total, err)
	}
	article.Status = "draft"
	if err := client.SyncArticle(ctx, article); err != nil {
		t.Fatal(err)
	}
	_, total, err = client.Search(ctx, content.ArticleFilter{Search: "数据库"}, 10, 0)
	if err != nil || total != 0 {
		t.Fatalf("draft should be removed: total=%d error=%v", total, err)
	}
}
