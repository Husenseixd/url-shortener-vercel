package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortURL string `json:"short_url"`
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateCode(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic bot protection
	userAgent := r.Header.Get("User-Agent")
	if isBot(userAgent) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Rate limiting for URL creation
	if !checkRateLimit(r) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		http.Error(w, "Redis not configured", http.StatusInternalServerError)
		return
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		http.Error(w, "Invalid Redis URL", http.StatusInternalServerError)
		return
	}
	rdb := redis.NewClient(opt)
	ctx := context.Background()

	var code string
	for {
		code = generateCode(6)
		exists, err := rdb.Exists(ctx, code).Result()
		if err != nil {
			http.Error(w, "Redis error", http.StatusInternalServerError)
			return
		}
		if exists == 0 {
			break
		}
	}

	if err := rdb.Set(ctx, code, req.URL, 0).Err(); err != nil {
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}

	// Log creation time
	rdb.Set(ctx, fmt.Sprintf("created:%s", code), time.Now().Format(time.RFC3339), 0)

	host := r.Host
	proto := "https://"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		proto = "http://"
	}
	resp := shortenResponse{ShortURL: fmt.Sprintf("%s%s/api/%s", proto, host, code)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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
