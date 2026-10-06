package assistant

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRateLimitKeyGroupsPortsWithoutStoringIPAddress(t *testing.T) {
	first := rateLimitKey("192.0.2.1:8080")
	if first != rateLimitKey("192.0.2.1:8081") || first == rateLimitKey("192.0.2.2:8080") {
		t.Fatal("key should share quota by IP, independently of port")
	}
	if first == "" || first == "devhub:assistant:limit:192.0.2.1" {
		t.Fatal("key should not expose the IP address")
	}
}

func TestRedisRateLimiterSharedQuota(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_REDIS") != "1" {
		t.Skip("set DEVHUB_TEST_REDIS=1 to test against local Redis")
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD")})
	defer client.Close()
	ctx := context.Background()
	remote := fmt.Sprintf("test-%d:8080", time.Now().UnixNano())
	key := rateLimitKey(remote)
	defer client.Del(ctx, key)
	first := NewRedisRateLimiter(client)
	second := NewRedisRateLimiter(client)
	for i := 0; i < 20; i++ {
		limiter := first
		if i%2 == 1 {
			limiter = second
		}
		allowed, err := limiter.Allow(ctx, remote)
		if err != nil || !allowed {
			t.Fatalf("request %d: allowed=%v error=%v", i+1, allowed, err)
		}
	}
	allowed, err := second.Allow(ctx, remote)
	if err != nil || allowed {
		t.Fatalf("21st request: allowed=%v error=%v", allowed, err)
	}
	if ttl := client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("expected expiring quota, TTL=%s", ttl)
	}
}
