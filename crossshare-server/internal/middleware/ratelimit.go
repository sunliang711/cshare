package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"crossshare-server/internal/db"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
	"crossshare-server/internal/settings"
)

type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
	enabled  bool
}

func NewRateLimiter(settingsMgr *settings.Manager) *RateLimiter {
	cur := settingsMgr.Get()
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
	rl.apply(cur.RateLimitEnable, cur.RateLimitRPM)
	settingsMgr.OnChange(func(next db.SystemSettings) {
		rl.apply(next.RateLimitEnable, next.RateLimitRPM)
	})
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) apply(enabled bool, rpm int) {
	if rpm <= 0 {
		rpm = 60
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	newR := rate.Limit(float64(rpm) / 60.0)
	if rl.r != newR || rl.burst != rpm {
		rl.limiters = make(map[string]*rate.Limiter)
	}
	rl.r = newR
	rl.burst = rpm
	rl.enabled = enabled
}

func (rl *RateLimiter) allow(ip string) (enabled bool, allowed bool) {
	rl.mu.RLock()
	enabled = rl.enabled
	if !enabled {
		rl.mu.RUnlock()
		return false, true
	}
	limiter, ok := rl.limiters[ip]
	rl.mu.RUnlock()

	if !ok {
		rl.mu.Lock()
		limiter, ok = rl.limiters[ip]
		if !ok {
			limiter = rate.NewLimiter(rl.r, rl.burst)
			rl.limiters[ip] = limiter
		}
		rl.mu.Unlock()
	}
	return true, limiter.Allow()
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		rl.limiters = make(map[string]*rate.Limiter)
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipRateLimit(c.Request.URL.Path) {
			c.Next()
			return
		}
		enabled, allowed := rl.allow(c.ClientIP())
		if !enabled {
			c.Next()
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(apperr.ErrRateLimitExceeded.HTTPStatus, model.Response{
				Code:      apperr.ErrRateLimitExceeded.Code,
				Msg:       apperr.ErrRateLimitExceeded.Message,
				Data:      nil,
				RequestID: c.GetString("request_id"),
			})
			return
		}
		c.Next()
	}
}

func skipRateLimit(path string) bool {
	switch path {
	case "/", "/style.css", "/app.js", "/qrcode-lite.js", "/favicon.svg":
		return true
	}
	if path == "/admin" || strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/assets/") {
		return true
	}
	return strings.HasPrefix(path, "/api/v1/p2p/sessions/") && strings.Contains(path, "/messages")
}
