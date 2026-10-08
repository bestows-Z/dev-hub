package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type oauthIdentity struct {
	ID        uint64    `json:"-"`
	UserID    uint64    `json:"-"`
	Provider  string    `json:"provider"`
	Subject   string    `json:"-"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (oauthIdentity) TableName() string { return "oauth_identities" }

var errEmailNeedsLink = errors.New("existing email requires explicit linking")
var errIdentityInUse = errors.New("provider identity already linked")
var errAccountDisabled = errors.New("account unavailable")

func (h *Handler) resolveOAuthUser(ctx context.Context, provider string, profile oauthProfile, linkUserID uint64) (*user.User, error) {
	var result user.User
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var identity oauthIdentity
		err := tx.Where("provider = ? AND subject = ?", provider, profile.subject).First(&identity).Error
		if err == nil {
			if linkUserID != 0 && identity.UserID != linkUserID {
				return errIdentityInUse
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, identity.UserID).Error; err != nil {
				return err
			}
			if result.Status != user.StatusNormal {
				return errAccountDisabled
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if linkUserID != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, linkUserID).Error; err != nil {
				return err
			}
			if result.Status != user.StatusNormal {
				return errAccountDisabled
			}
		} else {
			var count int64
			if err := tx.Model(&user.User{}).Where("email = ?", profile.email).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errEmailNeedsLink
			}
			random, err := oauthRandom()
			if err != nil {
				return err
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(random), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			name := []rune(strings.TrimSpace(profile.name))
			if len(name) > 60 {
				name = name[:60]
			}
			result = user.User{Username: provider + "_" + oauthHash(random)[:16], Email: profile.email, DisplayName: string(name), PasswordHash: string(hash), Role: user.RoleUser, Status: user.StatusNormal}
			if err := tx.Create(&result).Error; err != nil {
				return err
			}
		}
		identity = oauthIdentity{UserID: result.ID, Provider: provider, Subject: profile.subject, Email: profile.email}
		return tx.Create(&identity).Error
	})
	return &result, err
}
