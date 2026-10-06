package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter shares the assistant question quota across API instances.
type RateLimiter interface {
	Allow(ctx context.Context, remote string) (bool, error)
}

type RedisRateLimiter struct {
	client *redis.Client
}

var incrementWithExpiry = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`)

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, remote string) (bool, error) {
	count, err := incrementWithExpiry.Run(ctx, l.client, []string{rateLimitKey(remote)}, time.Minute.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return count <= 20, nil
}

func rateLimitKey(remote string) string {
	host := remote
	if parsed, _, err := net.SplitHostPort(remote); err == nil {
		host = parsed
	}
	host = strings.ToLower(strings.TrimSpace(host))
	digest := sha256.Sum256([]byte(host))
	return "devhub:assistant:limit:" + hex.EncodeToString(digest[:16])
}
