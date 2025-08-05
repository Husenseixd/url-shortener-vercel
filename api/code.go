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

func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/api/")
	if code == "" {
		http.NotFound(w, r)
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
	longURL, err := rdb.Get(ctx, code).Result()
	if err == redis.Nil {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Redis error", http.StatusInternalServerError)
		return
	}

	// Log the click
	logClick(ctx, rdb, code, r)

	http.Redirect(w, r, longURL, http.StatusFound)
}

func logClick(ctx context.Context, rdb *redis.Client, code string, r *http.Request) {
	// Increment click counter
	rdb.Incr(ctx, fmt.Sprintf("clicks:%s", code))

	// Increment total clicks
	rdb.Incr(ctx, "stats:total_clicks")

	// Increment today's clicks
	today := time.Now().Format("2006-01-02")
	rdb.Incr(ctx, fmt.Sprintf("stats:clicks:%s", today))

	// Update last click time
	rdb.Set(ctx, fmt.Sprintf("last_click:%s", code), time.Now().Format(time.RFC3339), 0)

	// Add to unique visitors set (last 24 hours)
	ip := getClientIP(r)
	rdb.SAdd(ctx, "stats:unique_visitors", ip)
	rdb.Expire(ctx, "stats:unique_visitors", 24*time.Hour)

	// Log click details
	clickLog := map[string]interface{}{
		"code":       code,
		"ip":         ip,
		"user_agent": r.Header.Get("User-Agent"),
		"referer":    r.Header.Get("Referer"),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(clickLog)
	rdb.ZAdd(ctx, "click_logs", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(logJSON),
	})

	// Keep only last 1000 click logs
	rdb.ZRemRangeByRank(ctx, "click_logs", 0, -1001)
}
