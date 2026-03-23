package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Logger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		evt := logger.Info()
		if c.Writer.Status() >= 500 {
			evt = logger.Error()
		} else if c.Writer.Status() >= 400 {
			evt = logger.Warn()
		}

		evt.
			Str("request_id", c.GetString("request_id")).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Int("size", c.Writer.Size()).
			Str("client_ip", c.ClientIP()).
			Msg("request")
	}
}
