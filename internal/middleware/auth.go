package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const userContextKey = "auth_user"

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(secret string, user models.AuthUser, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenValue string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenValue,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}

	return claims, nil
}

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
				"code":  "MISSING_AUTH_HEADER",
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header",
				"code":  "INVALID_AUTH_HEADER",
			})
			return
		}

		claims, err := ParseToken(secret, strings.TrimSpace(parts[1]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		c.Set(userContextKey, models.AuthUser{
			ID:       claims.Subject,
			Username: claims.Username,
			Role:     claims.Role,
		})
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
				"code":  "AUTH_REQUIRED",
			})
			return
		}
		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
				"code":  "ADMIN_REQUIRED",
			})
			return
		}

		c.Next()
	}
}

func CurrentUser(c *gin.Context) (models.AuthUser, bool) {
	value, ok := c.Get(userContextKey)
	if !ok {
		return models.AuthUser{}, false
	}

	user, ok := value.(models.AuthUser)
	return user, ok
}
