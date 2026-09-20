package middleware

import (
	appsecurity "FinancialPlannerGo/app/security"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type ReportRateLimiter struct {
	enabled    bool
	ratePerSec float64
	burst      float64
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	lastClean  time.Time
}

func NewReportRateLimiter(cfg appsecurity.Config) *ReportRateLimiter {
	return &ReportRateLimiter{
		enabled:    cfg.RateLimitEnabled,
		ratePerSec: float64(cfg.ReportRequestsPerMinute) / 60.0,
		burst:      float64(cfg.ReportRateLimitBurst),
		buckets:    make(map[string]*tokenBucket),
		lastClean:  time.Now(),
	}
}

func (l *ReportRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || !l.enabled {
			c.Next()
			return
		}

		key := c.ClientIP() + "|" + c.Request.Method + "|" + c.FullPath()
		if !l.allow(key) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, authErrorResponse{
				Success: false,
				Code:    "RATE_LIMITED",
				Message: "too many report requests; please try again later",
			})
			return
		}

		c.Next()
	}
}

func (l *ReportRateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: l.burst, lastRefill: now, lastSeen: now}
		l.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * l.ratePerSec
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.lastRefill = now
	bucket.lastSeen = now

	if now.Sub(l.lastClean) > 10*time.Minute {
		for bucketKey, candidate := range l.buckets {
			if now.Sub(candidate.lastSeen) > 30*time.Minute {
				delete(l.buckets, bucketKey)
			}
		}
		l.lastClean = now
	}

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}
