package project

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestRewriteRuntimeLocation(t *testing.T) {
	frontend := "/api/v1/project-runtimes/demo/"
	backend := frontend + "backend/"
	for _, test := range []struct {
		location string
		prefix   string
		want     string
	}{
		{"/login?next=%2Fhome", frontend, frontend + "login?next=%2Fhome"},
		{"/login", backend, backend + "login"},
		{"http://127.0.0.1:32769/docs", backend, backend + "docs"},
		{"https://example.com/docs", backend, "https://example.com/docs"},
		{"//example.com/docs", backend, "//example.com/docs"},
		{"login", backend, "login"},
	} {
		got := rewriteRuntimeLocation(test.location, test.prefix, "127.0.0.1:32769")
		if got != test.want {
			t.Errorf("location %q: got %q, want %q", test.location, got, test.want)
		}
	}
}

func TestRuntimeProxyAgainstMigratedPostgres(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_POSTGRES") != "1" {
		t.Skip("set DEVHUB_TEST_POSTGRES=1 to run the PostgreSQL integration test")
	}
	_ = godotenv.Load("../../../.env")
	t.Setenv("APP_ENV", "test")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	client, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/redirect") {
			w.Header().Set("Location", "/login")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Set-Cookie", "unsafe=1")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`))
	}))
	defer upstream.Close()
	parsed, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatal(err)
	}
	slug := fmt.Sprintf("runtime-proxy-test-%d", time.Now().UnixNano())
	rollback := errors.New("rollback test runtime")
	err = client.DB.Transaction(func(tx *gorm.DB) error {
		item := Project{Slug: slug, Title: "Runtime test", Tags: []string{}, Status: "published", RuntimeStatus: "running", RuntimeFrontendPort: port, RuntimeBackendPort: port}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Any("/api/v1/project-runtimes/:slug/*filepath", NewRuntimeHandler(tx, zap.NewNop()).Serve)
		publicServer := httptest.NewServer(router)
		defer publicServer.Close()
		httpClient := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		for _, test := range []struct{ path, location string }{
			{"/redirect", "/api/v1/project-runtimes/" + slug + "/login"},
			{"/backend/redirect", "/api/v1/project-runtimes/" + slug + "/backend/login"},
		} {
			response, err := httpClient.Get(publicServer.URL + "/api/v1/project-runtimes/" + slug + test.path)
			if err != nil {
				return err
			}
			_ = response.Body.Close()
			if response.StatusCode != http.StatusFound || response.Header.Get("Location") != test.location {
				return fmt.Errorf("redirect %s: status=%d location=%q", test.path, response.StatusCode, response.Header.Get("Location"))
			}
		}
		request, err := http.NewRequest(http.MethodGet, publicServer.URL+"/api/v1/project-runtimes/"+slug+"/script.svg", nil)
		if err != nil {
			return err
		}
		request.Header.Set("Cookie", "blog-session=secret")
		response, err := httpClient.Do(request)
		if err != nil {
			return err
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Security-Policy"), "sandbox") || response.Header.Get("Set-Cookie") != "" {
			return fmt.Errorf("svg response: status=%d CSP=%q Set-Cookie=%q", response.StatusCode, response.Header.Get("Content-Security-Policy"), response.Header.Get("Set-Cookie"))
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
