package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

const AdminCookieName = "crossshare_admin_token"

func AdminJWTSecret(cipher *crypto.Cipher) []byte {
	return cipher.DeriveSubKey("admin-jwt-signing")
}

func PasswordVersion(passwordHash string) string {
	sum := sha256.Sum256([]byte(passwordHash))
	return hex.EncodeToString(sum[:8])
}

func SignAdminToken(secret []byte, admin *db.Admin) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":      admin.ID,
		"username": admin.Username,
		"pwd_v":    PasswordVersion(admin.PasswordHash),
		"iat":      now.Unix(),
		"exp":      now.Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func AdminAuth(store *db.Store, secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		if cookieVal, err := c.Cookie(AdminCookieName); err == nil {
			tokenStr = cookieVal
		} else if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenStr == "" {
			abortAdminUnauthorized(c, "admin login required")
			return
		}

		tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		if err != nil || !tok.Valid {
			abortAdminUnauthorized(c, "invalid or expired admin session")
			return
		}

		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			abortAdminUnauthorized(c, "invalid admin token claims")
			return
		}

		adminID, _ := claims["sub"].(string)
		pwdV, _ := claims["pwd_v"].(string)
		admin, err := store.GetAdminByID(c.Request.Context(), adminID)
		if err != nil || !admin.IsActive {
			abortAdminUnauthorized(c, "admin account not active")
			return
		}
		if pwdV != PasswordVersion(admin.PasswordHash) {
			abortAdminUnauthorized(c, "admin session expired after password change")
			return
		}

		c.Set("admin", admin)
		c.Next()
	}
}

func CurrentAdmin(c *gin.Context) *db.Admin {
	if v, ok := c.Get("admin"); ok {
		if a, ok := v.(*db.Admin); ok {
			return a
		}
	}
	return nil
}

func abortAdminUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    "unauthorized",
			"message": msg,
		},
	})
}
