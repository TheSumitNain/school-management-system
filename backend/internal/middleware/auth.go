package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey   = "auth_user_id"
	ContextSchoolIDKey = "auth_school_id"
)

// AccessTokenValidator defines the token validation dependency
// required by the authentication middleware.
type AccessTokenValidator interface {
	ValidateAccessToken(tokenString string) (*services.TokenClaims, error)
}

// AuthMiddleware validates the Bearer access token and stores
// the authenticated user's identity in the Gin context.
func AuthMiddleware(tokenService AccessTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokenService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Authentication service unavailable",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		parts := strings.Fields(authHeader)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c)
			return
		}

		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil || claims == nil {
			unauthorized(c)
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || userID <= 0 || claims.SchoolID <= 0 {
			unauthorized(c)
			return
		}

		c.Set(ContextUserIDKey, userID)
		c.Set(ContextSchoolIDKey, claims.SchoolID)

		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"status":  "error",
		"message": "Authentication required",
	})
}
