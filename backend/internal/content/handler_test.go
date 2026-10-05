package content

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type fakeRepository struct {
	limit, offset int
	filter        ArticleFilter
}

func (r *fakeRepository) ListArticles(_ context.Context, filter ArticleFilter, limit, offset int) ([]Article, int64, error) {
	r.limit, r.offset = limit, offset
	r.filter = filter
	return []Article{{ID: 1, Slug: "hello", Title: "Hello", BodyMD: "private list body", Tags: []string{}}}, 1, nil
}
func (*fakeRepository) ArticleFacets(context.Context) (ArticleFacets, error) {
	return ArticleFacets{Categories: []ArticleFacet{}, Tags: []ArticleFacet{}}, nil
}
func (*fakeRepository) GetArticle(context.Context, string) (*Article, error) {
	return nil, ErrArticleNotFound
}
func (*fakeRepository) ListLinks(context.Context) ([]FriendLink, error) { return []FriendLink{}, nil }

func TestListArticlesPaginatesAndOmitsBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepository{}
	router := gin.New()
	router.GET("/articles", NewHandler(repo, zap.NewNop()).ListArticles)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/articles?page=2&page_size=100&category=travel&tag=%E6%97%85%E8%A1%8C&q=hello", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if repo.limit != 50 || repo.offset != 50 {
		t.Fatalf("pagination limit=%d offset=%d", repo.limit, repo.offset)
	}
	if repo.filter.Category != "travel" || repo.filter.Tag != "旅行" || repo.filter.Search != "hello" {
		t.Fatalf("filter %+v", repo.filter)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	item := payload["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
	if _, ok := item["body_md"]; ok {
		t.Fatal("list leaked article body")
	}
}

func TestGetArticleReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/articles/:slug", NewHandler(&fakeRepository{}, zap.NewNop()).GetArticle)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/articles/draft", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}
