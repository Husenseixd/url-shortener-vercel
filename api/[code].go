package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func handler(w http.ResponseWriter, r *http.Request) {
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
	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	http.HandleFunc("/api/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
