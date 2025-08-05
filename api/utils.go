package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func isBot(userAgent string) bool {
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "curl", "wget",
		"python", "java", "perl", "ruby", "php", "go-http-client",
	}

	userAgent = strings.ToLower(userAgent)
	for _, pattern := range botPatterns {
		if strings.Contains(userAgent, pattern) {
			return true
		}
	}
	return false
}

func checkRateLimit(r *http.Request) bool {
	// Simple rate limiting - in production, use a proper rate limiter
	// This is a basic implementation
	ip := getClientIP(r)
	key := fmt.Sprintf("rate_limit:%s", ip)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return true // Allow if Redis not configured
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return true
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()

	// Check current count
	count, err := rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return true
	}

	if count > 100 { // 100 requests per minute
		return false
	}

	// Increment counter
	rdb.Incr(ctx, key)
	rdb.Expire(ctx, key, time.Minute)

	return true
}

func getClientIP(r *http.Request) string {
	// Get real IP from various headers
	headers := []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
	for _, header := range headers {
		if ip := r.Header.Get(header); ip != "" {
			return strings.Split(ip, ",")[0]
		}
	}
	return r.RemoteAddr
} 