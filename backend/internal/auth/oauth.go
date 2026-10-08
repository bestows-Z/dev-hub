package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/http/response"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const oauthCookiePath = "/api/v1/auth/oauth"

type oauthService struct {
	providers         map[string]oauthProvider
	redis             *redis.Client
	client            *http.Client
	publicURL, webURL string
	secure            bool
}

type oauthFlow struct {
	Provider   string
	Verifier   string
	LinkUserID uint64
}
type oauthTicket struct {
	UserID uint64
	Proof  string
}

func oauthRandom() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
func oauthHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) SetOAuth(cfg config.OAuthConfig, client *redis.Client, appEnv string) error {
	if (cfg.GitHubClientID == "") != (cfg.GitHubClientSecret == "") || (cfg.GoogleClientID == "") != (cfg.GoogleClientSecret == "") {
		return errors.New("OAuth client id and secret must both be configured")
	}
	if cfg.GitHubClientID == "" && cfg.GoogleClientID == "" {
		h.oauth = nil
		return nil
	}
	public, err := url.Parse(cfg.PublicURL)
	web, webErr := url.Parse(cfg.WebURL)
	valid := func(u *url.URL) bool {
		return u != nil && (u.Scheme == "https" || ((appEnv == "local" || appEnv == "test") && u.Scheme == "http")) && u.Host != "" && u.User == nil && (u.Path == "" || u.Path == "/") && u.RawQuery == "" && u.Fragment == ""
	}
	if err != nil || webErr != nil || !valid(public) || !valid(web) || public.Scheme != web.Scheme || !strings.EqualFold(public.Hostname(), web.Hostname()) {
		return errors.New("AUTH_PUBLIC_URL and WEB_PUBLIC_URL must be valid origins sharing a hostname; use HTTPS outside local development")
	}
	providers := map[string]oauthProvider{
		"github": {id: cfg.GitHubClientID, secret: cfg.GitHubClientSecret, authorizeURL: "https://github.com/login/oauth/authorize", tokenURL: "https://github.com/login/oauth/access_token", profileURL: "https://api.github.com/user", emailURL: "https://api.github.com/user/emails", scope: "read:user user:email"},
		"google": {id: cfg.GoogleClientID, secret: cfg.GoogleClientSecret, authorizeURL: "https://accounts.google.com/o/oauth2/v2/auth", tokenURL: "https://oauth2.googleapis.com/token", profileURL: "https://openidconnect.googleapis.com/v1/userinfo", scope: "openid email profile"},
	}
	for name, p := range providers {
		if (p.id == "") != (p.secret == "") {
			return fmt.Errorf("%s OAuth client id and secret must both be configured", name)
		}
	}
	h.oauth = &oauthService{providers: providers, redis: client, publicURL: strings.TrimRight(cfg.PublicURL, "/"), webURL: strings.TrimRight(cfg.WebURL, "/"), secure: public.Scheme == "https", client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	return nil
}

func (s *oauthService) callbackURL(provider string) string {
	return s.publicURL + oauthCookiePath + "/" + provider + "/callback"
}
func (s *oauthService) cookie(c *gin.Context, name, value string, seconds int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: value, Path: oauthCookiePath, MaxAge: seconds, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
}
func (h *Handler) OAuthProviders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	items := []gin.H{}
	for _, name := range []string{"github", "google"} {
		items = append(items, gin.H{"id": name, "enabled": h.oauth != nil && h.oauth.providers[name].id != ""})
	}
	response.Success(c, items)
}

func (h *Handler) beginOAuth(c *gin.Context, linkUserID uint64) (string, error) {
	s := h.oauth
	name := c.Param("provider")
	if s == nil || s.providers[name].id == "" {
		return "", errors.New("provider disabled")
	}
	count, err := s.redis.Eval(c.Request.Context(), `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],3600) end; return n`, []string{"oauth:limit:" + oauthHash(c.ClientIP())}).Int()
	if err != nil || count > 30 {
		return "", errors.New("OAuth request limit reached")
	}
	state, err := oauthRandom()
	if err != nil {
		return "", err
	}
	verifier, err := oauthRandom()
	if err != nil {
		return "", err
	}
	flow, _ := json.Marshal(oauthFlow{Provider: name, Verifier: verifier, LinkUserID: linkUserID})
	if err = s.redis.Set(c.Request.Context(), "oauth:state:"+oauthHash(state), flow, 10*time.Minute).Err(); err != nil {
		return "", err
	}
	s.cookie(c, "devhub_oauth_state", state, 600)
	sum := sha256.Sum256([]byte(verifier))
	p := s.providers[name]
	query := url.Values{"client_id": {p.id}, "redirect_uri": {s.callbackURL(name)}, "response_type": {"code"}, "scope": {p.scope}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}}
	return p.authorizeURL + "?" + query.Encode(), nil
}

func (h *Handler) StartOAuth(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	authorize, err := h.beginOAuth(c, 0)
	if err != nil {
		h.oauthFailure(c, "unavailable")
		return
	}
	c.Redirect(http.StatusFound, authorize)
}
func (h *Handler) LinkOAuth(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	authorize, err := h.beginOAuth(c, CurrentUser(c).ID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50311, Message: "第三方登录暂未开放或暂时不可用。"})
		return
	}
	response.Success(c, gin.H{"url": authorize})
}
func (h *Handler) oauthFailure(c *gin.Context, reason string) {
	if h.oauth == nil {
		c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50311, Message: "第三方登录暂时不可用。"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, h.oauth.webURL+"/auth/callback?error="+url.QueryEscape(reason))
}

func (h *Handler) OAuthCallback(c *gin.Context) {
	s := h.oauth
	if s == nil {
		h.oauthFailure(c, "unavailable")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	state := c.Query("state")
	cookie, err := c.Cookie("devhub_oauth_state")
	if err != nil || len(state) != 43 || subtle.ConstantTimeCompare([]byte(state), []byte(cookie)) != 1 {
		h.oauthFailure(c, "expired")
		return
	}
	data, err := s.redis.GetDel(c.Request.Context(), "oauth:state:"+oauthHash(state)).Bytes()
	s.cookie(c, "devhub_oauth_state", "", -1)
	var flow oauthFlow
	if err != nil || json.Unmarshal(data, &flow) != nil || flow.Provider != c.Param("provider") {
		h.oauthFailure(c, "expired")
		return
	}
	if c.Query("error") != "" {
		h.oauthFailure(c, "cancelled")
		return
	}
	code := c.Query("code")
	if code == "" || len(code) > 2048 {
		h.oauthFailure(c, "expired")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	profile, err := s.profile(ctx, flow.Provider, code, flow.Verifier)
	if errors.Is(err, errUnverifiedEmail) {
		h.oauthFailure(c, "email")
		return
	}
	if err != nil {
		h.logger.Warn("OAuth provider authentication failed", zap.String("provider", flow.Provider))
		h.oauthFailure(c, "provider")
		return
	}
	u, err := h.resolveOAuthUser(ctx, flow.Provider, profile, flow.LinkUserID)
	if err != nil {
		var dbErr *pgconn.PgError
		if errors.Is(err, errEmailNeedsLink) || (errors.As(err, &dbErr) && dbErr.ConstraintName == "uq_users_email") {
			h.oauthFailure(c, "link_required")
			return
		}
		if errors.Is(err, errIdentityInUse) || (errors.As(err, &dbErr) && (dbErr.ConstraintName == "uq_oauth_subject" || dbErr.ConstraintName == "uq_oauth_user_provider")) {
			h.oauthFailure(c, "identity_in_use")
			return
		}
		if errors.Is(err, errAccountDisabled) {
			h.oauthFailure(c, "account")
			return
		}
		h.logger.Error("OAuth identity persistence failed", zap.String("provider", flow.Provider))
		h.oauthFailure(c, "provider")
		return
	}
	if flow.LinkUserID != 0 {
		c.Redirect(http.StatusFound, s.webURL+"/auth/callback?linked=1")
		return
	}
	ticket, err := oauthRandom()
	if err != nil {
		h.oauthFailure(c, "provider")
		return
	}
	proof, err := oauthRandom()
	if err != nil {
		h.oauthFailure(c, "provider")
		return
	}
	payload, _ := json.Marshal(oauthTicket{UserID: u.ID, Proof: oauthHash(proof)})
	if err := s.redis.Set(ctx, "oauth:ticket:"+oauthHash(ticket), payload, time.Minute).Err(); err != nil {
		h.oauthFailure(c, "provider")
		return
	}
	s.cookie(c, "devhub_oauth_proof", proof, 60)
	c.Redirect(http.StatusFound, s.webURL+"/auth/callback?ticket="+url.QueryEscape(ticket))
}

func (h *Handler) ExchangeOAuth(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var input struct {
		Ticket string `json:"ticket" binding:"required,len=43"`
	}
	if c.ShouldBindJSON(&input) != nil || h.oauth == nil {
		response.Fail(c, 40103, "登录授权已过期，请重新发起第三方登录。")
		return
	}
	s := h.oauth
	proof, err := c.Cookie("devhub_oauth_proof")
	if err != nil {
		response.Fail(c, 40103, "登录授权已过期，请重新发起第三方登录。")
		return
	}
	key := "oauth:ticket:" + oauthHash(input.Ticket)
	data, err := s.redis.Get(c.Request.Context(), key).Bytes()
	var ticket oauthTicket
	if err != nil || json.Unmarshal(data, &ticket) != nil || subtle.ConstantTimeCompare([]byte(ticket.Proof), []byte(oauthHash(proof))) != 1 {
		response.Fail(c, 40103, "登录授权已过期，请重新发起第三方登录。")
		return
	}
	// Only a caller with the browser proof may consume this one-time ticket.
	if _, err := s.redis.GetDel(c.Request.Context(), key).Result(); err != nil {
		response.Fail(c, 40103, "登录授权已过期，请重新发起第三方登录。")
		return
	}
	s.cookie(c, "devhub_oauth_proof", "", -1)
	u, err := h.users.FindByID(c.Request.Context(), ticket.UserID)
	if err != nil || u.Status != user.StatusNormal {
		response.Fail(c, 40102, "账户不可用，请联系站长。")
		return
	}
	h.completeLogin(c, u)
}

func (h *Handler) OAuthIdentities(c *gin.Context) {
	items := []oauthIdentity{}
	if err := h.db.WithContext(c.Request.Context()).Where("user_id = ?", CurrentUser(c).ID).Order("created_at").Find(&items).Error; err != nil {
		response.Error(c)
		return
	}
	response.Success(c, items)
}
func (h *Handler) UnlinkOAuth(c *gin.Context) {
	name := c.Param("provider")
	if name != "github" && name != "google" {
		response.Fail(c, response.CodeInvalidParams, "无效的第三方账号。")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Where("user_id = ? AND provider = ?", CurrentUser(c).ID, name).Delete(&oauthIdentity{}).Error; err != nil {
		response.Error(c)
		return
	}
	response.Success(c, gin.H{"removed": true})
}
