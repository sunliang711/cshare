package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"crossshare-server/internal/config"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
)

type RateLimiter struct {
	limiters sync.Map
	r        rate.Limit
	burst    int
	enabled  bool
}

func NewRateLimiter(cfg *config.Config) *RateLimiter {
	rpm := cfg.RateLimit.RequestsPerMinute
	if rpm <= 0 {
		rpm = 60
	}
	rl := &RateLimiter{
		r:       rate.Limit(float64(rpm) / 60.0),
		burst:   rpm,
		enabled: cfg.RateLimit.Enable,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	if v, ok := rl.limiters.Load(ip); ok {
		return v.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rl.r, rl.burst)
	actual, _ := rl.limiters.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.limiters.Range(func(key, _ interface{}) bool {
			rl.limiters.Delete(key)
			return true
		})
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
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
