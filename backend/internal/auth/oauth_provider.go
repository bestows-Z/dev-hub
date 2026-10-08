package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
)

type oauthProvider struct {
	id, secret, authorizeURL, tokenURL, profileURL, emailURL, scope string
}

type oauthProfile struct{ subject, email, name string }

var errUnverifiedEmail = errors.New("provider has no verified email")

// Provider endpoints are fixed by the server, never supplied by an HTTP request.
func (s *oauthService) fetchJSON(ctx context.Context, method, endpoint, token string, form url.Values, output any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DevHub")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.New("provider connection failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return errors.New("provider response is unavailable or oversized")
	}
	return json.Unmarshal(data, output)
}

func (s *oauthService) profile(ctx context.Context, providerName, code, verifier string) (oauthProfile, error) {
	p := s.providers[providerName]
	var tokens struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}
	err := s.fetchJSON(ctx, http.MethodPost, p.tokenURL, "", url.Values{
		"client_id": {p.id}, "client_secret": {p.secret}, "code": {code},
		"redirect_uri": {s.callbackURL(providerName)}, "grant_type": {"authorization_code"}, "code_verifier": {verifier},
	}, &tokens)
	if err != nil || tokens.Error != "" || tokens.AccessToken == "" || !strings.EqualFold(tokens.TokenType, "bearer") {
		return oauthProfile{}, errors.New("provider token exchange failed")
	}
	var profile oauthProfile
	if providerName == "github" {
		var account struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Login string `json:"login"`
		}
		if err := s.fetchJSON(ctx, http.MethodGet, p.profileURL, tokens.AccessToken, nil, &account); err != nil {
			return profile, err
		}
		if account.ID <= 0 {
			return profile, errors.New("invalid provider identity")
		}
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := s.fetchJSON(ctx, http.MethodGet, p.emailURL, tokens.AccessToken, nil, &emails); err != nil {
			return profile, err
		}
		for _, email := range emails {
			if email.Primary && email.Verified {
				profile.email = email.Email
				break
			}
		}
		profile.subject, profile.name = strconv.FormatInt(account.ID, 10), account.Name
		if profile.name == "" {
			profile.name = account.Login
		}
	} else {
		var account struct {
			Subject  string `json:"sub"`
			Email    string `json:"email"`
			Name     string `json:"name"`
			Verified bool   `json:"email_verified"`
		}
		if err := s.fetchJSON(ctx, http.MethodGet, p.profileURL, tokens.AccessToken, nil, &account); err != nil {
			return profile, err
		}
		if account.Verified {
			profile.email = account.Email
		}
		profile.subject, profile.name = account.Subject, account.Name
	}
	profile.email = strings.ToLower(strings.TrimSpace(profile.email))
	address, err := mail.ParseAddress(profile.email)
	if err != nil || address.Address != profile.email || len(profile.email) > 255 || strings.ContainsAny(profile.email, "\r\n ") {
		return profile, errUnverifiedEmail
	}
	if profile.subject == "" || len(profile.subject) > 255 {
		return profile, errors.New("invalid provider identity")
	}
	return profile, nil
}
