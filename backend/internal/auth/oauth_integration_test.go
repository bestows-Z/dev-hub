package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestOAuthFlowIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_OAUTH") != "1" {
		t.Skip("set DEVHUB_TEST_OAUTH=1 for local PostgreSQL and Redis")
	}
	_ = godotenv.Load("../../../.env")
	t.Setenv("APP_ENV", "test")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
	defer rdb.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback OAuth test")
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		suffix := uuid.NewString()
		providerID := time.Now().UnixNano()
		email := "oauth-" + suffix + "@example.invalid"
		h := NewHandler(user.NewRepository(tx), NewTokens(cfg.Auth.JWTSecret), tx, nil, zap.NewNop())
		if err := h.SetOAuth(config.OAuthConfig{PublicURL: "http://localhost:5173", WebURL: "http://localhost:5173", GitHubClientID: "test-id", GitHubClientSecret: "test-secret"}, rdb, "test"); err != nil {
			return err
		}
		expectedChallenge := ""
		tokenCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/token":
				_ = r.ParseForm()
				tokenCalls++
				sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
				if base64.RawURLEncoding.EncodeToString(sum[:]) != expectedChallenge {
					t.Error("PKCE challenge mismatch")
				}
				fmt.Fprint(w, `{"access_token":"test-token","token_type":"bearer"}`)
			case "/profile":
				fmt.Fprintf(w, `{"id":%d,"name":"Test reader"}`, providerID)
			case "/emails":
				fmt.Fprintf(w, `[{"email":%q,"primary":true,"verified":true}]`, email)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		provider := h.oauth.providers["github"]
		provider.tokenURL = server.URL + "/token"
		provider.profileURL = server.URL + "/profile"
		provider.emailURL = server.URL + "/emails"
		h.oauth.providers["github"] = provider
		router := gin.New()
		router.GET("/oauth/:provider/start", h.StartOAuth)
		router.GET("/oauth/:provider/callback", h.OAuthCallback)
		router.POST("/exchange", h.ExchangeOAuth)
		request := func(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
			r.Header.Set("Content-Type", "application/json")
			for _, cookie := range cookies {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			return w
		}
		start := request("GET", "/oauth/github/start", "")
		if start.Code != 302 {
			return fmt.Errorf("start failed: %d", start.Code)
		}
		authorize, _ := url.Parse(start.Header().Get("Location"))
		state := authorize.Query().Get("state")
		expectedChallenge = authorize.Query().Get("code_challenge")
		if len(state) != 43 || expectedChallenge == "" || authorize.Query().Get("code_challenge_method") != "S256" {
			return errors.New("missing state or PKCE")
		}
		stateCookie := start.Result().Cookies()[0]
		if !stateCookie.HttpOnly || stateCookie.SameSite != http.SameSiteLaxMode {
			return errors.New("unsafe state cookie")
		}
		callbackPath := "/oauth/github/callback?state=" + url.QueryEscape(state) + "&code=test-code"
		wrong := request("GET", callbackPath, "", &http.Cookie{Name: stateCookie.Name, Value: "wrong-browser"})
		if !strings.Contains(wrong.Header().Get("Location"), "error=expired") {
			return errors.New("accepted callback without browser state")
		}
		callback := request("GET", callbackPath, "", stateCookie)
		location, _ := url.Parse(callback.Header().Get("Location"))
		ticket := location.Query().Get("ticket")
		if len(ticket) != 43 || strings.Contains(location.RawQuery, "access_token") {
			return fmt.Errorf("callback did not issue opaque ticket: %s", location.Path)
		}
		var proofCookie *http.Cookie
		for _, cookie := range callback.Result().Cookies() {
			if cookie.Name == "devhub_oauth_proof" {
				proofCookie = cookie
			}
		}
		if proofCookie == nil || !proofCookie.HttpOnly {
			return errors.New("missing browser proof")
		}
		body := fmt.Sprintf(`{"ticket":%q}`, ticket)
		wrongExchange := request("POST", "/exchange", body, &http.Cookie{Name: proofCookie.Name, Value: "wrong"})
		var envelope struct {
			Code int
			Data struct {
				AccessToken string        `json:"access_token"`
				User        user.Response `json:"user"`
			}
		}
		if json.Unmarshal(wrongExchange.Body.Bytes(), &envelope) != nil || envelope.Code == 0 {
			return errors.New("accepted ticket from wrong browser")
		}
		exchanged := request("POST", "/exchange", body, proofCookie)
		if json.Unmarshal(exchanged.Body.Bytes(), &envelope) != nil || envelope.Code != 0 || envelope.Data.User.Role != user.RoleUser {
			return fmt.Errorf("exchange failed: %s", exchanged.Body.String())
		}
		id, err := h.tokens.Verify(envelope.Data.AccessToken)
		if err != nil || id != envelope.Data.User.ID {
			return errors.New("bad session token")
		}
		repeat := request("POST", "/exchange", body, proofCookie)
		if json.Unmarshal(repeat.Body.Bytes(), &envelope) != nil || envelope.Code == 0 {
			return errors.New("ticket was reusable")
		}
		replay := request("GET", callbackPath, "", stateCookie)
		if !strings.Contains(replay.Header().Get("Location"), "error=expired") || tokenCalls != 1 {
			return errors.New("state was reusable")
		}
		// A known email may be linked only after an authenticated user initiates it.
		profile := oauthProfile{subject: suffix, email: email, name: "Another identity"}
		if _, err := h.resolveOAuthUser(ctx, "google", profile, 0); !errors.Is(err, errEmailNeedsLink) {
			return fmt.Errorf("email was automatically merged: %v", err)
		}
		if _, err := h.resolveOAuthUser(ctx, "google", profile, id); err != nil {
			return fmt.Errorf("explicit link failed: %w", err)
		}
		if _, err := h.resolveOAuthUser(ctx, "google", profile, id+1); !errors.Is(err, errIdentityInUse) {
			return fmt.Errorf("identity could be taken over: %v", err)
		}
		if err := tx.Model(&user.User{}).Where("id = ?", id).Update("status", user.StatusDisabled).Error; err != nil {
			return err
		}
		if _, err := h.resolveOAuthUser(ctx, "google", profile, 0); !errors.Is(err, errAccountDisabled) {
			return fmt.Errorf("disabled user accepted: %v", err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
