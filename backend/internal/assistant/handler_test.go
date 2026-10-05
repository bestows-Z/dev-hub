package assistant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
)

func TestQueryTermsFindsChineseTopicAndLatinName(t *testing.T) {
	terms := queryTerms("你写过哪些关于数据库和 PostgreSQL 的文章？")
	joined := strings.Join(terms, ",")
	if !strings.Contains(joined, "数据库") || !strings.Contains(joined, "postgresql") {
		t.Fatalf("missing important terms: %v", terms)
	}
}

func TestBestPassageUsesRelevantParagraph(t *testing.T) {
	article := content.Article{
		Slug:   "postgres-notes",
		Title:  "数据库笔记",
		BodyMD: "这段讲页面设计。\n\nPostgreSQL 使用事务确保库存更新。\n\n最后是部署记录。",
	}
	passage := bestPassage(article, []string{"postgresql", "库存"})
	if !strings.Contains(passage.Text, "库存") || passage.URL != "/articles/postgres-notes" {
		t.Fatalf("wrong passage: %+v", passage)
	}
}

func TestTruncateDoesNotSplitChineseRune(t *testing.T) {
	if got := truncate("你好世界", 3); got != "你好世…" {
		t.Fatalf("truncate = %q", got)
	}
}

func TestGenerateSendsArticleContextWithoutLeakingKeyToClient(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected model request: path=%s authorization=%s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) != 2 || !strings.Contains(request.Messages[1].Content, "库存更新") {
			t.Fatalf("article passage was not provided: %+v", request.Messages)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"库存通过事务更新。"}}]}`)), Header: make(http.Header)}, nil
	})
	handler := &Handler{cfg: config.AssistantConfig{APIBaseURL: "http://localhost/v1", APIKey: "test-key", Model: "test-model"}, client: &http.Client{Transport: transport}}
	answer, err := handler.generate(context.Background(), "库存怎么更新？", []passage{{Source: Source{Title: "库存笔记", URL: "/articles/stock"}, Text: "库存更新使用事务。"}})
	if err != nil || answer != "库存通过事务更新。" {
		t.Fatalf("answer = %q, err = %v", answer, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestRateLimitResetsAfterMinute(t *testing.T) {
	handler := &Handler{usage: make(map[string]requestUsage)}
	now := time.Now()
	for i := 0; i < 20; i++ {
		if !handler.allow("127.0.0.1:8080", now) {
			t.Fatal("request blocked too early")
		}
	}
	if handler.allow("127.0.0.1:8080", now) || !handler.allow("127.0.0.1:8080", now.Add(time.Minute+time.Second)) {
		t.Fatal("rate limit did not block and reset correctly")
	}
}
