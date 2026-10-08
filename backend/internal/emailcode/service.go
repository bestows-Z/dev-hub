package emailcode

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTooFrequent = errors.New("verification code requested too frequently")

type Service struct {
	redis  *redis.Client
	sender Sender
	secret []byte
}

func NewService(client *redis.Client, sender Sender, secret string) *Service {
	return &Service{redis: client, sender: sender, secret: []byte(secret)}
}

func (s *Service) key(kind, purpose, email string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(kind + "|" + purpose + "|" + strings.ToLower(strings.TrimSpace(email))))
	return "email-code:" + kind + ":" + hex.EncodeToString(h.Sum(nil))
}

func (s *Service) digest(purpose, email, code string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(purpose + "|" + strings.ToLower(strings.TrimSpace(email)) + "|" + code))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Service) Request(ctx context.Context, purpose, email, ip string) error {
	if purpose != "register" && purpose != "login" && purpose != "change_email" {
		return fmt.Errorf("invalid email code purpose")
	}
	codeNumber, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", codeNumber.Int64())
	cooldown := s.key("cooldown", purpose, email)
	claimed, err := s.redis.SetNX(ctx, cooldown, "1", time.Minute).Result()
	if err != nil {
		return err
	}
	if !claimed {
		return ErrTooFrequent
	}
	ipKey := s.key("ip-limit", purpose, ip)
	requests, err := s.redis.Incr(ctx, ipKey).Result()
	if err != nil {
		_ = s.redis.Del(ctx, cooldown).Err()
		return err
	}
	if requests == 1 {
		_ = s.redis.Expire(ctx, ipKey, time.Hour).Err()
	}
	if requests > 20 {
		_ = s.redis.Del(ctx, cooldown).Err()
		return ErrTooFrequent
	}
	codeKey := s.key("value", purpose, email)
	_, err = s.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, codeKey, s.digest(purpose, email, code), 10*time.Minute)
		pipe.Del(ctx, s.key("attempts", purpose, email))
		return nil
	})
	if err != nil {
		_ = s.redis.Del(ctx, cooldown).Err()
		return err
	}
	if err := s.sender.Send(ctx, email, code, purpose); err != nil {
		_ = s.redis.Del(ctx, codeKey, cooldown).Err()
		return err
	}
	return nil
}

const verifyScript = `
local actual = redis.call('GET', KEYS[1])
if not actual then return 0 end
if actual == ARGV[1] then
  redis.call('DEL', KEYS[1], KEYS[2])
  return 1
end
local attempts = redis.call('INCR', KEYS[2])
if attempts == 1 then redis.call('EXPIRE', KEYS[2], 600) end
if attempts >= 5 then redis.call('DEL', KEYS[1]) end
return 0
`

func (s *Service) Verify(ctx context.Context, purpose, email, code string) (bool, error) {
	if len(code) != 6 {
		return false, nil
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false, nil
		}
	}
	result, err := s.redis.Eval(ctx, verifyScript, []string{s.key("value", purpose, email), s.key("attempts", purpose, email)}, s.digest(purpose, email, code)).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
