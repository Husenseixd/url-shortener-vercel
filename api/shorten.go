package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func generateCode(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	host := r.Host
	proto := "https://"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		proto = "http://"
	}
	resp := shortenResponse{ShortURL: fmt.Sprintf("%s%s/api/%s", proto, host, code)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/api/shorten", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
