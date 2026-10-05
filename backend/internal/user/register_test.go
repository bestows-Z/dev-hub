package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type registerRepository struct {
	created        *User
	usernameExists bool
	emailExists    bool
}

func (r *registerRepository) ExistsByUsername(context.Context, string) (bool, error) {
	return r.usernameExists, nil
}
func (r *registerRepository) ExistsByEmail(context.Context, string) (bool, error) {
	return r.emailExists, nil
}
func (r *registerRepository) Create(_ context.Context, u *User) error {
	r.created = u
	u.ID = 7
	return nil
}
func (r *registerRepository) FindByIdentifier(context.Context, string) (*User, error) {
	return nil, ErrNotFound
}
func (r *registerRepository) FindByID(context.Context, uint64) (*User, error) {
	return nil, ErrNotFound
}

func TestRegisterCanonicalizesAndHashesPassword(t *testing.T) {
	repo := &registerRepository{}
	got, err := NewService(repo).Register(context.Background(), RegisterRequest{
		Username: " Alice_1 ", Email: " ALICE@example.com ", Password: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice_1" || got.Email != "alice@example.com" || got.ID != 7 {
		t.Fatalf("unexpected public user: %+v", got)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.created.PasswordHash), []byte("secret123")); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterRejectsDuplicateAndLongPassword(t *testing.T) {
	if _, err := NewService(&registerRepository{usernameExists: true}).Register(context.Background(), RegisterRequest{"alice", "a@example.com", "secret123"}); !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("got %v", err)
	}
	if _, err := NewService(&registerRepository{}).Register(context.Background(), RegisterRequest{"alice", "a@example.com", strings.Repeat("a", 73)}); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("got %v", err)
	}
}
