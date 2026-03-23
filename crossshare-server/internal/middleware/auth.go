package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"crossshare-server/internal/config"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
)

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Auth.Enable {
			c.Next()
			return
		}

		headerName := cfg.Auth.JWTHeaderName
		if headerName == "" {
			headerName = "Authorization"
		}

		authHeader := c.GetHeader(headerName)
		if authHeader == "" {
			abortWithAppError(c, apperr.ErrAuthInvalid)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			abortWithAppError(c, apperr.ErrAuthInvalid)
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			abortWithAppError(c, apperr.ErrAuthInvalid)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(string); ok {
				c.Set("creator", sub)
			}
		}

		c.Next()
	}
}

func abortWithAppError(c *gin.Context, e *apperr.AppError) {
	c.AbortWithStatusJSON(e.HTTPStatus, model.Response{
		Code:      e.Code,
		Msg:       e.Message,
		Data:      nil,
		RequestID: c.GetString("request_id"),
	})
}
