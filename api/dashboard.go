package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type DashboardStats struct {
	TotalURLs      int64 `json:"total_urls"`
	TotalClicks    int64 `json:"total_clicks"`
	TodayClicks    int64 `json:"today_clicks"`
	UniqueVisitors int64 `json:"unique_visitors"`
}

type URLLog struct {
	Code      string    `json:"code"`
	LongURL   string    `json:"long_url"`
	Clicks    int64     `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
	LastClick time.Time `json:"last_click"`
}

type ClickLog struct {
	Code      string    `json:"code"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Timestamp time.Time `json:"timestamp"`
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic bot protection - check for common bot user agents
	userAgent := r.Header.Get("User-Agent")
	if isBot(userAgent) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Rate limiting
	if !checkRateLimit(r) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
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

	// Get dashboard stats
	stats, err := getDashboardStats(ctx, rdb)
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	// Get recent URL logs
	urlLogs, err := getRecentURLLogs(ctx, rdb, 50)
	if err != nil {
		http.Error(w, "Failed to get URL logs", http.StatusInternalServerError)
		return
	}

	// Get recent click logs
	clickLogs, err := getRecentClickLogs(ctx, rdb, 100)
	if err != nil {
		http.Error(w, "Failed to get click logs", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"stats":      stats,
		"url_logs":   urlLogs,
		"click_logs": clickLogs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

func getDashboardStats(ctx context.Context, rdb *redis.Client) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Get total URLs
	totalURLs, err := rdb.Keys(ctx, "url:*").Result()
	if err != nil {
		return nil, err
	}
	stats.TotalURLs = int64(len(totalURLs))

	// Get total clicks
	totalClicks, err := rdb.Get(ctx, "stats:total_clicks").Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats.TotalClicks = totalClicks

	// Get today's clicks
	today := time.Now().Format("2006-01-02")
	todayClicks, err := rdb.Get(ctx, fmt.Sprintf("stats:clicks:%s", today)).Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats.TodayClicks = todayClicks

	// Get unique visitors (last 24 hours)
	uniqueVisitors, err := rdb.SCard(ctx, "stats:unique_visitors").Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats.UniqueVisitors = uniqueVisitors

	return stats, nil
}

func getRecentURLLogs(ctx context.Context, rdb *redis.Client, limit int) ([]URLLog, error) {
	// Get all URL keys
	keys, err := rdb.Keys(ctx, "url:*").Result()
	if err != nil {
		return nil, err
	}

	var logs []URLLog
	for _, key := range keys {
		if len(logs) >= limit {
			break
		}

		code := strings.TrimPrefix(key, "url:")
		longURL, err := rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		// Get click count
		clicks, err := rdb.Get(ctx, fmt.Sprintf("clicks:%s", code)).Int64()
		if err != nil && err != redis.Nil {
			clicks = 0
		}

		// Get creation time
		createdAt := time.Now() // Default to now if not stored
		if createdStr, err := rdb.Get(ctx, fmt.Sprintf("created:%s", code)).Result(); err == nil {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				createdAt = t
			}
		}

		// Get last click time
		lastClick := time.Time{} // Zero time if no clicks
		if lastClickStr, err := rdb.Get(ctx, fmt.Sprintf("last_click:%s", code)).Result(); err == nil {
			if t, err := time.Parse(time.RFC3339, lastClickStr); err == nil {
				lastClick = t
			}
		}

		logs = append(logs, URLLog{
			Code:      code,
			LongURL:   longURL,
			Clicks:    clicks,
			CreatedAt: createdAt,
			LastClick: lastClick,
		})
	}

	return logs, nil
}

func getRecentClickLogs(ctx context.Context, rdb *redis.Client, limit int) ([]ClickLog, error) {
	// Get recent click logs from sorted set
	logs, err := rdb.ZRevRange(ctx, "click_logs", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	var clickLogs []ClickLog
	for _, logStr := range logs {
		var log ClickLog
		if err := json.Unmarshal([]byte(logStr), &log); err != nil {
			continue
		}
		clickLogs = append(clickLogs, log)
	}

	return clickLogs, nil
}
