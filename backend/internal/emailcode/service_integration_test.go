package emailcode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type captureSender struct {
	code       string
	err        error
	duringSend func()
}

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	_ = godotenv.Load("../../../.env")
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		address = "127.0.0.1:6379"
	}
	return redis.NewClient(&redis.Options{Addr: address, Password: os.Getenv("REDIS_PASSWORD")})
}

func (s *captureSender) Send(_ context.Context, _, code, _ string) error {
	s.code = code
	if s.duringSend != nil {
		s.duringSend()
	}
	return s.err
}

func TestCodeLifecycleIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_REDIS") != "1" {
		t.Skip("set DEVHUB_TEST_REDIS=1 to test against local Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := testRedis(t)
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	sender := &captureSender{}
	service := NewService(client, sender, "test-secret-unique-for-email-codes")
	email := fmt.Sprintf("test-%s@example.invalid", uuid.NewString())
	ip := uuid.NewString()
	if err := service.Request(ctx, "register", email, ip); err != nil {
		t.Fatal(err)
	}
	if len(sender.code) != 6 {
		t.Fatalf("expected six-digit code, got %q", sender.code)
	}
	if err := service.Request(ctx, "register", email, ip); !errors.Is(err, ErrTooFrequent) {
		t.Fatalf("expected resend cooldown, got %v", err)
	}
	wrongCode := "999999"
	if sender.code == wrongCode {
		wrongCode = "000000"
	}
	for attempt := 0; attempt < 5; attempt++ {
		valid, err := service.Verify(ctx, "register", email, wrongCode)
		if err != nil || valid {
			t.Fatalf("wrong code accepted on attempt %d: %v", attempt+1, err)
		}
	}
	valid, err := service.Verify(ctx, "register", email, sender.code)
	if err != nil || valid {
		t.Fatalf("code should be invalidated after five attempts: %v", err)
	}
	if err := client.Del(ctx, service.key("cooldown", "register", email)).Err(); err != nil {
		t.Fatal(err)
	}
	// Simulate a verification arriving while SMTP is delivering a replacement code.
	if err := client.Set(ctx, service.key("attempts", "register", email), 4, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	sender.duringSend = func() {
		wrong := "999999"
		if sender.code == wrong {
			wrong = "000000"
		}
		if valid, err := service.Verify(ctx, "register", email, wrong); err != nil || valid {
			t.Fatalf("unexpected verification during delivery: %v", err)
		}
	}
	if err := service.Request(ctx, "register", email, ip); err != nil {
		t.Fatal(err)
	}
	valid, err = service.Verify(ctx, "register", email, sender.code)
	if err != nil || !valid {
		t.Fatalf("new code should reset attempts: %v", err)
	}
	valid, err = service.Verify(ctx, "register", email, sender.code)
	if err != nil || valid {
		t.Fatalf("code should be single use: %v", err)
	}
}

func TestSenderFailureDoesNotReserveCodeIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_REDIS") != "1" {
		t.Skip("set DEVHUB_TEST_REDIS=1 to test against local Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := testRedis(t)
	defer client.Close()
	sender := &captureSender{err: errors.New("SMTP unavailable")}
	service := NewService(client, sender, "test-secret-unique-for-email-codes")
	email := fmt.Sprintf("test-%s@example.invalid", uuid.NewString())
	if err := service.Request(ctx, "login", email, uuid.NewString()); err == nil {
		t.Fatal("expected sender error")
	}
	sender.err = nil
	if err := service.Request(ctx, "login", email, uuid.NewString()); err != nil {
		t.Fatalf("retry after sender recovery failed: %v", err)
	}
}
