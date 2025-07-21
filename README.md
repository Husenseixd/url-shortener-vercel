# URL Shortener (Go + Vercel + Redis)

A simple URL shortener web app built with Go, deployable on Vercel, using Redis for storage. No user accounts, just paste a link and get a short URL!

## Features
- Shorten any URL with one click
- Fast redirects
- Simple, modern web UI
- Powered by Go serverless functions (Vercel)
- Uses Redis for storage (Upstash or similar)

## Usage
1. Deploy to Vercel
2. Set `REDIS_URL` environment variable
3. Visit the site, paste your long URL, and click Shorten

## Project Structure
- `/api/shorten.go` - API endpoint to create short URLs
- `/api/[code].go` - API endpoint to redirect short URLs
- `/index.html` - Frontend web page

---
MIT License 