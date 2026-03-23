package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"crossshare-server/internal/config"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: cfg.CORS.AllowOrigins,
		AllowMethods: cfg.CORS.AllowMethods,
		AllowHeaders: cfg.CORS.AllowHeaders,
		ExposeHeaders: []string{
			"Crossshare-Type",
			"Crossshare-Filename",
			"Key-Deleted",
			"Content-Type",
			"X-Request-Id",
		},
		AllowCredentials: true,
	})
}
