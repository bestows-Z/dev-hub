package user

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type recordingVerifier struct{ calls int }

func (v *recordingVerifier) Verify(context.Context, string, string, string) (bool, error) {
	v.calls++
	return true, nil
}

func TestInvalidRegistrationPreservesCode(t *testing.T) {
	for _, test := range []struct {
		name     string
		username string
		password string
		exists   bool
	}{
		{"invalid username", "a-b", "secret123", false},
		{"duplicate username", "alice", "secret123", true},
		{"long password", "alice", strings.Repeat("a", 73), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier := &recordingVerifier{}
			h := NewHandler(NewService(&registerRepository{usernameExists: test.exists}), zap.NewNop())
			h.SetEmailVerifier(verifier)
			r := gin.New()
			r.POST("/register", h.Register)
			body, _ := json.Marshal(map[string]string{"username": test.username, "password": test.password, "email": "alice@example.com", "email_code": "123456"})
			request := httptest.NewRequest("POST", "/register", strings.NewReader(string(body)))
			request.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(httptest.NewRecorder(), request)
			if verifier.calls != 0 {
				t.Fatal("invalid form consumed the verification code")
			}
		})
	}
}
