package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware returns a Gin middleware that validates a JWT Bearer token.
// It expects the JWT secret as argument; if empty, it will try to read from env JWT_SECRET.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	if jwtSecret == "" {
		jwtSecret = os.Getenv("JWT_SECRET")
	}

	return func(c *gin.Context) {
		if jwtSecret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}

		ah := c.GetHeader("Authorization")
		if ah == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(ah, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// Ensure signing method is HMAC
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenUnverifiable
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Extract expected claims and put them into Gin context
		if sub, ok := claims["sub"].(string); ok {
			c.Set("staff_id", sub)
		}
		if hid, ok := claims["hospital_id"].(string); ok {
			c.Set("hospital_id", hid)
		}
		if uname, ok := claims["username"].(string); ok {
			c.Set("username", uname)
		}
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}

		c.Next()
	}
}
