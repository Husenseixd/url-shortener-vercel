# URL Shortener (Go + Vercel + Redis)

A simple URL shortener web app built with Go, deployable on Vercel, using Redis for storage. Features a public dashboard with analytics and bot protection.

## Features
- Shorten any URL with one click
- Fast redirects
- **Public Dashboard** with real-time analytics
- **Bot Protection** with rate limiting and user agent filtering
- **User Activity Logging** with IP tracking and click analytics
- **Modern Web UI** with responsive design
- Powered by Go serverless functions (Vercel)
- Uses Redis for storage (Upstash or similar)

## Dashboard Features
- **Real-time Statistics**: Total URLs, clicks, today's clicks, unique visitors
- **URL Analytics**: View all shortened URLs with click counts and timestamps
- **Click Logs**: Detailed logs of every click with IP, user agent, and referer
- **Auto-refresh**: Dashboard updates every 30 seconds
- **Mobile Responsive**: Works on all devices

## Bot Protection
- **User Agent Filtering**: Blocks common bot user agents
- **Rate Limiting**: 100 requests per minute per IP
- **IP Tracking**: Real IP detection from various headers
- **Access Control**: Prevents automated access to dashboard

## Usage
1. Deploy to Vercel
2. Set `REDIS_URL` environment variable
3. Visit the site, paste your long URL, and click Shorten
4. Access the dashboard at `/dashboard.html` to view analytics

## Project Structure
- `/api/shorten.go` - API endpoint to create short URLs
- `/api/code.go` - API endpoint to redirect short URLs with logging
- `/api/dashboard.go` - Dashboard API with analytics and bot protection
- `/index.html` - Main frontend web page
- `/dashboard.html` - Dashboard frontend with analytics

## Environment Variables
- `REDIS_URL` - Redis connection string (required)

---
MIT License 