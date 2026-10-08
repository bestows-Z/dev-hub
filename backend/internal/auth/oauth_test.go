package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
)

func TestOAuthConfiguration(t *testing.T) {
	h := &Handler{}
	if err := h.SetOAuth(config.OAuthConfig{}, nil, "production"); err != nil || h.oauth != nil {
		t.Fatalf("disabled OAuth must not prevent startup: %v", err)
	}
	if err := h.SetOAuth(config.OAuthConfig{GitHubClientID: "only-id"}, nil, "local"); err == nil {
		t.Fatal("accepted incomplete credentials")
	}
	if err := h.SetOAuth(config.OAuthConfig{GitHubClientID: "id", GitHubClientSecret: "secret", PublicURL: "http://localhost:5173", WebURL: "http://localhost:5173"}, nil, "production"); err == nil {
		t.Fatal("accepted HTTP production origins")
	}
}

func TestOAuthProviderVerifiedEmail(t *testing.T) {
	for _, test := range []struct {
		name, provider, profile, emails string
		valid                           bool
	}{
		{"GitHub verified primary", "github", `{"id":42,"login":"reader"}`, `[{"email":"reader@example.com","primary":true,"verified":true}]`, true},
		{"GitHub unverified", "github", `{"id":42,"login":"reader"}`, `[{"email":"reader@example.com","primary":true,"verified":false}]`, false},
		{"Google verified", "google", `{"sub":"stable-subject","email":"reader@example.com","email_verified":true}`, "", true},
		{"Google unverified", "google", `{"sub":"stable-subject","email":"reader@example.com","email_verified":false}`, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/token":
					_ = r.ParseForm()
					if r.Form.Get("client_secret") != "test-secret" || r.Form.Get("code_verifier") != "test-verifier" {
						t.Error("token request missing secret or PKCE verifier")
					}
					fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer"}`)
				case "/profile":
					if r.Header.Get("Authorization") != "Bearer test-token" {
						t.Error("profile missing authorization")
					}
					fmt.Fprint(w, test.profile)
				case "/emails":
					fmt.Fprint(w, test.emails)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			s := &oauthService{client: server.Client(), publicURL: "http://localhost:5173", providers: map[string]oauthProvider{test.provider: {id: "test-id", secret: "test-secret", tokenURL: server.URL + "/token", profileURL: server.URL + "/profile", emailURL: server.URL + "/emails"}}}
			profile, err := s.profile(context.Background(), test.provider, "test-code", "test-verifier")
			if test.valid && (err != nil || profile.email != "reader@example.com" || profile.subject == "") {
				t.Fatalf("profile = %+v, err=%v", profile, err)
			}
			if !test.valid && !errors.Is(err, errUnverifiedEmail) {
				t.Fatalf("unverified email accepted: %v", err)
			}
		})
	}
}
