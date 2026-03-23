package middleware

import (
	"crossshare-server/internal/keygen"

	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			generated, _ := keygen.Generate(12)
			rid = "req-" + generated
		}
		c.Set("request_id", rid)
		c.Header("X-Request-Id", rid)
		c.Next()
	}
}
